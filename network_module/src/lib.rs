//! # Network Module
//!
//! Simulates a WiFi ad-hoc network between UAVs (Unmanned Aerial Vehicles).
//!
//! The simulator models realistic wireless communication constraints:
//! - **Transmission time**: calculated from payload size.
//! - **Busy sender detection**: a UAV that is currently transmitting cannot send another message.
//! - **Spatial collision avoidance**: if nearby UAVs (within the same spatial chunk) are
//!   transmitting, new transmissions are delayed.
//! - **Signal overlap (collision)**: overlapping transmissions at the receiver are dropped.
//! - **Distance-based packet loss**: probability of packet loss increases with distance
//!   between sender and receiver.
//!
//! ## Architecture
//!
//! The broker receives UDP datagrams from UAV clients, which can be either:
//! - **Position updates**: register or update a UAV's position and address.
//! - **Data messages**: enqueue a message from one UAV to another.
//!
//! Messages are processed in cycles controlled by [`NetworkSimulator::send_messages`],
//! which flushes enqueued messages via UDP and retries any delayed messages.
//!
//! ## Message Formats
//!
//! All UDP datagrams are JSON-encoded. The broker distinguishes between the two
//! types by the presence of the `target_id` field.
//!
//! ### Position Update
//!
//! Registers or updates a UAV's position and address. Sent without `target_id`.
//!
//! ```json
//! {
//!   "sender_id": "drone_0",
//!   "position": { "x": 40.4168, "y": -3.7038, "z": 0.1 }
//! }
//! ```
//!
//! ### Data Message
//!
//! Sends a payload from one UAV to another. Requires `target_id` and `payload`.
//!
//! ```json
//! {
//!   "sender_id": "drone_0",
//!   "target_id": "drone_1",
//!   "payload": [72, 101, 108, 108, 111]
//! }
//! ```

use std::{cmp::{self}, collections::{HashMap, VecDeque}, mem, net::{SocketAddr, UdpSocket}, sync::Arc, thread, time::{Duration, Instant}};

use crate::models::{Message, Position, UAV};
use crate::logger::Logger;

pub mod models;
pub mod logger;

/// Iteration order for checking adjacent spatial chunks.
/// Starts with the origin chunk (0), then checks neighbors (-1, +1).
const CHUNK_ORDER: [i64; 3] = [0, -1, 1];

/// Size of each spatial chunk in kilometers.
const CHUNK_SIZE: f64 = 1.1;

/// Approximate kilometers per degree of geographic coordinates.
const KM_PER_COORDS_DEGREE: f64 = 111.12;

/// Conversion factor from geographic coordinates to chunk indices.
const KM_TO_CHUNK_COORD: f64 = KM_PER_COORDS_DEGREE / CHUNK_SIZE;

/// Core network simulator that manages UAV registration, message enqueuing,
/// collision detection, and UDP delivery.
///
/// # Workflow
///
/// 1. UAVs register themselves via [`update_uav_info`](Self::update_uav_info).
/// 2. Messages are enqueued via [`enqueue_message`](Self::enqueue_message), which
///    checks sender availability, spatial collisions, and distance-based loss.
/// 3. [`send_messages`](Self::send_messages) is called periodically to flush
///    enqueued messages and retry delayed ones.
pub struct NetworkSimulator {
  /// Registered UAVs indexed by their string ID.
  uavs: HashMap<String, UAV>,
  /// Messages waiting to be retried (delayed due to busy sender or nearby transmissions).
  delayed_msgs: VecDeque<Message>,
  /// Messages ready to be sent, grouped by target UAV ID.
  sended_msgs: HashMap<String, Vec<Message>>,
  /// Tracks active transmissions per spatial chunk for collision detection.
  /// Key is the 3D chunk coordinate, value is a list of `busy_until` instants.
  uavs_sending: HashMap<(i64, i64, i64), Vec<Instant>>,
  /// Shared UDP socket used to deliver messages to UAVs.
  socket: Arc<UdpSocket>,
  /// Shared logger instance for console output.
  logger: Arc<Logger>,
}

impl NetworkSimulator {
  /// Creates a new `NetworkSimulator` bound to the given UDP port.
  ///
  /// # Panics
  ///
  /// Panics if the UDP socket cannot be bound to `0.0.0.0:{port}`.
  pub fn new(port: &str) -> Self {
    Self {
      uavs: HashMap::new(),
      delayed_msgs: VecDeque::new(),
      sended_msgs: HashMap::new(),
      uavs_sending: HashMap::new(),
      socket: Arc::new(UdpSocket::bind(format!("0.0.0.0:{}", port)).expect("couldn't bind to address")),
      logger: Arc::new(Logger::new()),
    }
  }

  /// Returns a reference to the UAV with the given ID, or `None` if not registered.
  pub fn get_uav(&self, uav_id: &str) -> Option<&UAV> {
    self.uavs.get(uav_id)
  }

  /// Registers a new UAV or updates an existing one's position and address.
  ///
  /// If the UAV ID doesn't exist, a new entry is created with `busy_until` set to now.
  pub fn update_uav_info(&mut self, uav_id: String, position: Position, addr: SocketAddr) {
    let uav = self.uavs.entry(uav_id.clone()).or_insert(UAV{
      position: Position::default(),
      busy_until: Instant::now(),
      addr: addr,
    });
    uav.position = position;
    uav.addr = addr;
    self.logger.uav_registered(&uav_id);
  }

  /// Flushes all enqueued messages via UDP and retries delayed messages.
  ///
  /// For each target UAV, messages are checked for signal overlap (collision):
  /// if two messages' transmission windows overlap at the receiver, the overlapped
  /// message is dropped. Non-overlapped messages are sent via UDP in a background thread.
  ///
  /// After sending, any delayed messages are re-enqueued for another attempt.
  pub fn send_messages(&mut self) {
    let socket = self.socket.clone();
    let logger = self.logger.clone();
    if !self.sended_msgs.is_empty() {
      let sended_msgs = mem::take(&mut self.sended_msgs);

      thread::spawn(move || {
        for (target_id, msgs) in &sended_msgs {
          let mut send_next = true;
          let mut msg = &msgs[0];
          let mut max_end = msg.to;

          for i in 1..msgs.len() {
            let next_msg = &msgs[i];
            let overlapped = next_msg.from <= max_end;
            
            if !overlapped && send_next {
              Self::send_udp(&socket, msg, &logger);
            } else if overlapped {
              logger.msg_overlapped(target_id);
            }

            msg = next_msg;
            max_end = cmp::max(msg.to, max_end);
            send_next = !overlapped;
          }

          if send_next {
            Self::send_udp(&socket, msg, &logger);
          }
        }
      });
      
      self.sended_msgs = HashMap::new();
    }

    let delayed_count = self.delayed_msgs.len();
    self.logger.delayed_queue_retry(delayed_count);

    let mut delayed_msgs = self.delayed_msgs.clone();
    while let Some(msg) = delayed_msgs.pop_front() {
      self.enqueue_message(msg.sender_id, msg.target_id, msg.payload);
    }

    self.delayed_msgs.clear();
  }

  /// Enqueues a message from `sender_id` to `target_id` with the given payload.
  ///
  /// The message goes through several checks before being enqueued:
  /// 1. **Sender existence**: if the sender is not registered, the message is discarded.
  /// 2. **Sender busy**: if the sender is still transmitting, the message is delayed.
  /// 3. **Nearby senders**: if other UAVs in adjacent spatial chunks are transmitting,
  ///    the message is delayed (collision avoidance).
  /// 4. **Target existence**: if the target is not registered, the message is discarded.
  /// 5. **Distance check**: probabilistic packet loss based on sender-target distance.
  ///
  /// If all checks pass, the message is added to the send queue and the sender's
  /// `busy_until` is updated based on the transmission time.
  pub fn enqueue_message(&mut self, sender_id: String, target_id: String, payload: Vec<u8>){
    let (sender_pos, sender_busy_until) = {
      if let Some(sender) = self.uavs.get(&sender_id) {
        (sender.position.clone(), sender.busy_until)
      } else {
        self.logger.msg_discarded_unknown_sender(&sender_id, &target_id);
        return;
      }
    };
    
    let now = Instant::now();

    let sender_busy = now < sender_busy_until;
    let near_senders = self.is_near_senders(&sender_pos, &now);

    if sender_busy || near_senders {
      if sender_busy {
        self.logger.msg_delayed_sender_busy(&sender_id, &target_id);
      } else {
        self.logger.msg_delayed_near_senders(&sender_id, &target_id);
      }
      self.delayed_msgs.push_back(Message {
        sender_id: sender_id.clone(),
        target_id: target_id.clone(),
        payload,
        from: now,
        ..Default::default()
      });
      return;
    }

    if let Some(target) = self.uavs.get(&target_id) {
      if !self.pass_distance_check(&sender_pos, &target.position) {
        self.logger.msg_discarded_distance(&sender_id, &target_id);
        return;
      }
    } else {
      self.logger.msg_discarded_unknown_target(&sender_id, &target_id);
      return;
    }

    let tx = (payload.len() as u64) * 8 / 6;
    let busy_until = now + Duration::from_micros(tx);
    
    let coords_key = self.get_coords_key(&sender_pos);
    let pos_based_uavs_sending = self.uavs_sending.entry(coords_key.clone()).or_insert(Vec::new());
    pos_based_uavs_sending.push(busy_until.clone());

    if let Some(sender) = self.uavs.get_mut(&sender_id) {
      sender.busy_until = busy_until;
    }
    
    if let Some(target) = self.uavs.get_mut(&target_id) {
      let entry = self.sended_msgs.entry(target_id.clone()).or_insert(Vec::new());
      target.busy_until = busy_until;
      entry.push(Message {
        sender_id: sender_id.clone(), 
        target_id: target_id.clone(), 
        target_addr: Some(target.addr),
        payload: payload.clone(), 
        from: now,
        to: busy_until,
        tx,
        ..Default::default()
      });
      self.logger.msg_enqueued(&sender_id, &target_id, payload.len(), tx);
    }
  }

  /// Sends a message payload via UDP to the target's address.
  fn send_udp(socket: &UdpSocket, msg: &Message, logger: &Logger) {
    let request = socket.send_to(&msg.payload, msg.target_addr.unwrap());
    match request {
      Err(e) => {
        logger.msg_sent_err(&msg.target_id, &e.to_string());
      }
      Ok(_) => {
        logger.msg_sent_ok(&msg.target_id);
      }
    }
  }

  /// Converts a geographic position into discrete 3D chunk coordinates.
  ///
  /// Each chunk represents a `CHUNK_SIZE` km region of space.
  fn get_coords_key(&self, pos: &Position) -> (i64, i64, i64) {
    (
      (pos.x * KM_TO_CHUNK_COORD).floor() as i64,
      (pos.y * KM_TO_CHUNK_COORD).floor() as i64,
      (pos.z * KM_TO_CHUNK_COORD).floor() as i64,
    )
  }

  /// Checks whether any UAV in the same or adjacent spatial chunks is currently transmitting.
  ///
  /// Iterates over the 27 neighboring chunks (3×3×3 grid) and checks if any
  /// sender's `busy_until` instant is still in the future. Expired entries are
  /// drained from the list for cleanup.
  ///
  /// Returns `true` if at least one nearby sender is still active.
  fn is_near_senders(&mut self, uav_pos: &Position, now: &Instant) -> bool {
    let simplified_coords = self.get_coords_key(uav_pos);

    for x in CHUNK_ORDER {
      for y in CHUNK_ORDER {
        for z in CHUNK_ORDER {
          let key = (
            simplified_coords.0 + x,
            simplified_coords.1 + y,
            simplified_coords.2 + z,
          );

          if let Some(senders) = self.uavs_sending.get_mut(&key) {
            if senders.is_empty() {
              self.uavs_sending.remove(&key);
              continue;
            }

            for (i, busy_until) in senders.iter().enumerate() {
              if now > busy_until { continue }

              senders.drain(..i);
              return true;
            }
          }
        }
      }
    }

    false
  }

  /// Probabilistic distance-based packet loss check.
  ///
  /// Calculates the Euclidean distance between sender and receiver positions
  /// and derives a loss probability using a quadratic model:
  /// `loss_prob = 0.0000005335 * d² + 0.00003395 * d`
  ///
  /// Returns `true` if the packet survives (random value exceeds loss probability).
  fn pass_distance_check(&self, sender_pos: &Position, receiver_pos: &Position) -> bool {
    let dx = receiver_pos.x - sender_pos.x;
    let dy = receiver_pos.y - sender_pos.y;
    let dz = receiver_pos.z - sender_pos.z;
    let d = (dx * dx + dy * dy + dz * dz).sqrt();

    let loss_prob = 0.0000005335 * d * d + 0.00003395 * d;
    let rand_val: f64 = rand::random();

    rand_val > loss_prob
  }

  /// Returns a reference to the delayed messages queue.
  pub fn get_delayed_msgs(&self) -> &VecDeque<Message> {
    &self.delayed_msgs
  }

  /// Returns a reference to the enqueued (ready to send) messages, grouped by target ID.
  pub fn get_sended_msgs(&self) -> &HashMap<String, Vec<Message>> {
    &self.sended_msgs
  }

  /// Returns a reference to the active transmissions map (chunk coords → busy_until instants).
  pub fn get_uavs_sending(&self) -> &HashMap<(i64, i64, i64), Vec<Instant>> {
    &self.uavs_sending
  }

  /// Returns a reference to all registered UAVs.
  pub fn get_uavs(&self) -> &HashMap<String, UAV> {
    &self.uavs
  }
}