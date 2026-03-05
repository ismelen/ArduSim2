use std::{cmp::{self}, collections::{HashMap, VecDeque}, mem, net::{SocketAddr, UdpSocket}, sync::Arc, thread, time::{Duration, Instant}};

use crate::models::{Message, Position, UAV};
use crate::logger::Logger;

pub mod models;
pub mod logger;

const CHUNK_ORDER: [i64; 3] = [0, -1, 1];
const CHUNK_SIZE: f64 = 1.1;
const KM_PER_COORDS_DEGREE: f64 = 111.12;
const KM_TO_CHUNK_COORD: f64 = KM_PER_COORDS_DEGREE / CHUNK_SIZE;

pub struct NetworkSimulator {
  uavs: HashMap<String, UAV>,
  delayed_msgs: VecDeque<Message>,
  sended_msgs: HashMap<String, Vec<Message>>,
  uavs_sending: HashMap<(i64, i64, i64), Vec<Instant>>,
  socket: Arc<UdpSocket>,
  logger: Arc<Logger>,
}

impl NetworkSimulator {
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

  pub fn get_uav(&self, uav_id: &str) -> Option<&UAV> {
    self.uavs.get(uav_id)
  }

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

  fn get_coords_key(&self, pos: &Position) -> (i64, i64, i64) {
    (
      (pos.x * KM_TO_CHUNK_COORD).floor() as i64,
      (pos.y * KM_TO_CHUNK_COORD).floor() as i64,
      (pos.z * KM_TO_CHUNK_COORD).floor() as i64,
    )
  }


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

  fn pass_distance_check(&self, sender_pos: &Position, receiver_pos: &Position) -> bool {
    let dx = receiver_pos.x - sender_pos.x;
    let dy = receiver_pos.y - sender_pos.y;
    let dz = receiver_pos.z - sender_pos.z;
    let d = (dx * dx + dy * dy + dz * dz).sqrt();

    let loss_prob = 0.0000005335 * d * d + 0.00003395 * d;
    let rand_val: f64 = rand::random();

    rand_val > loss_prob
  }

  pub fn get_delayed_msgs(&self) -> &VecDeque<Message> {
    &self.delayed_msgs
  }

  pub fn get_sended_msgs(&self) -> &HashMap<String, Vec<Message>> {
    &self.sended_msgs
  }

  pub fn get_uavs_sending(&self) -> &HashMap<(i64, i64, i64), Vec<Instant>> {
    &self.uavs_sending
  }

  pub fn get_uavs(&self) -> &HashMap<String, UAV> {
    &self.uavs
  }
}