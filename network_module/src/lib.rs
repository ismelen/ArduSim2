use std::{cmp::{self, Ordering}, collections::{HashMap, VecDeque}, mem, net::{SocketAddr, UdpSocket}, sync::{Arc, RwLock}, thread, time::{Duration, Instant}};

use serde::Deserialize;

#[derive(Clone, Debug, Default, Deserialize)]
pub struct Position {
  pub x: f64,
  pub y: f64,
  pub z: f64,
}

#[derive(Clone, Debug)]
pub struct UAV {
  position: Position,
  busy_until: Instant,
  addr: SocketAddr,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Message {
  sender_id: String,
  target_id: String,
  target_addr: Option<SocketAddr>,
  payload: Vec<u8>,
  from: Instant,
  to: Instant,
  tx: u64, // Transmission time in microseconds
  overlapped: bool,
}

impl Ord for Message {
  fn cmp(&self, other: &Self) -> Ordering {
      self.from.cmp(&other.from)
  }
}

impl PartialOrd for Message {
  fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
    self.from.partial_cmp(&other.from)
  }
}

impl Default for Message {
    fn default() -> Self {
      Self { 
        sender_id: Default::default(), 
        target_id: Default::default(), 
        payload: Default::default(), 
        from: Instant::now(), 
        to: Instant::now(),
        tx: 0,
        overlapped: false,
        target_addr: None,
      }
    }
}

pub struct SenderInfo {
  position: Position,
  sending_until: Instant
}

impl Default for SenderInfo {
  fn default() -> Self {
    Self{
      sending_until: Instant::now(),
      position: Position::default()
    }
  }
}

pub struct NetworkSimulator {
  uavs: HashMap<String, UAV>,
  delayed_msgs: VecDeque<Message>,
  sended_msgs: HashMap<String, Vec<Message>>,
  uavs_sending: Vec<SenderInfo>,
  socket: Arc<UdpSocket>,
}

impl NetworkSimulator {
  pub fn new(port: &str) -> Self {
    Self {
      uavs: HashMap::new(),
      delayed_msgs: VecDeque::new(),
      sended_msgs: HashMap::new(),
      uavs_sending: Vec::new(),
      socket: Arc::new(UdpSocket::bind(format!("0.0.0.0:{}", port)).expect("couldn't bind to address"))
    }
  }

  pub fn update_uav_info(&mut self, uav_id: String, position: Position, addr: SocketAddr) {
    let uav = self.uavs.entry(uav_id).or_insert(UAV{
      position: Position::default(),
      busy_until: Instant::now(),
      addr: addr,
    });
    uav.position = position;
    uav.addr = addr;
  }

  pub fn send_messages(&mut self) {
    let socket = self.socket.clone();
    let sended_msgs = mem::take(&mut self.sended_msgs);

    thread::spawn(move || {
      //TODO: Comprobar tambien si está cerca de los senders
      for (_, msgs) in &sended_msgs {
        let mut send_next = true;
        let mut msg = &msgs[0];
        let mut max_end = msg.to;

        for i in 1..msgs.len() {
          let next_msg = &msgs[i];
          let overlapped = next_msg.from <= max_end;
          
          if !overlapped && send_next {
            Self::send_udp(&socket, msg);
          }

          msg = next_msg;
          max_end = cmp::max(msg.to, max_end);
          send_next = !overlapped;
        }

        if send_next {
          Self::send_udp(&socket, msg);
        }
      }
    });

    self.sended_msgs = HashMap::new();

    let mut delayed_msgs = self.delayed_msgs.clone();
    while let Some(msg) = delayed_msgs.pop_front() {
      self.enqueue_message(msg.sender_id, msg.target_id, msg.payload);
    }

    self.delayed_msgs.clear();
  }

  pub fn enqueue_message(&mut self, sender_id: String, target_id: String, payload: Vec<u8>){
    let sender = self.uavs.get(&sender_id);
    if sender.is_none() {
      return; // Sender UAV not found, ignore message
    } 
    let sender = sender.unwrap();
    let now = Instant::now();

    if now < sender.busy_until || self.is_near_senders(&sender.position, &now){
      self.delayed_msgs.push_back(Message {
        sender_id: sender_id.clone(),
        target_id: target_id.clone(),
        payload,
        from: now,
        ..Default::default()
      });
      return; // Sender is busy, delay the message
    }

    if let Some(target) = self.uavs.get(&target_id) {
      if !self.pass_distance_check(&sender.position, &target.position) {
        return; // Message lost due to distance, ignore
      }
    }
    
    let entry = self.sended_msgs.entry(target_id.clone()).or_insert(Vec::new());
    let tx = (payload.len() as u64) * 8 / 6;
    let busy_until = now + Duration::from_micros(tx);
    
    self.uavs_sending.push(SenderInfo { position: sender.position.clone(), sending_until: busy_until });

    if let Some(sender) = self.uavs.get_mut(&sender_id) {
      sender.busy_until = busy_until;
    }
    if let Some(target) = self.uavs.get_mut(&target_id) {
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
    }
  }

  fn send_udp(socket: &UdpSocket, msg: &Message) {
    let request = socket.send_to(&msg.payload, msg.target_addr.unwrap());
    match request {
      Err(e) => {
        println!("{}", e)
      }
      Ok(_) => {  
        println!("Success!")
      }
    }
  }

  fn is_near_senders(&self, uav_pos: &Position, now: &Instant) -> bool {
    
    
    for sender in self.uavs_sending.iter() {
      if *now > sender.sending_until {
        continue;
      }

      let dx = sender.position.x - uav_pos.x;
      let dy = sender.position.y - uav_pos.y;
      let dz = sender.position.z - uav_pos.z;
      let distance = (dx*dx + dy*dy + dz*dz).sqrt();

      if distance < 1100.0 {
        return true
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
}