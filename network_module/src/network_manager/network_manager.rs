use std::{net::UdpSocket, sync::mpsc, thread, time::{Duration, SystemTime}};
use tokio_stream::StreamExt;
use tokio_util::time::DelayQueue;
use crate::network_manager::{models::{BusyState, Message}, uav_manager::UAVManager};


pub struct NetworkManager {
  uav_manager: UAVManager,
  msg_processor_chan: Option<mpsc::Sender<Message>>,
  coords_chan: Option<mpsc::Sender<Message>>,
  sender_chan: Option<tokio::sync::mpsc::Sender<(Message, Duration)>>,
}

impl NetworkManager {
  pub fn new() -> Self {
    let manager = UAVManager::new();

    let mut nm = NetworkManager{
      uav_manager: manager,
      coords_chan: None,
      sender_chan: None,
      msg_processor_chan: None,
    };

    nm.start_coords_thread();
    nm.start_sender_thread();
    nm.start_message_processor_thread();

    nm
  }

  pub fn push(&self, data: serde_json::Value) {
    let msg: Message = match serde_json::from_value(data) {
      Ok(msg) => msg,
      Err(_) => {
        println!("Cannot parse json message");
        return;
      }
    };

    if msg.receiver_id.is_none() && msg.coords.is_some() {
      match self.coords_chan {
        Some(ref chan) => {
          chan.send(msg).unwrap();
        },
        None => {
          println!("No coords channel available");
        }
      }
      return
    }

    match self.msg_processor_chan {
      Some(ref chan) => {
        chan.send(msg).unwrap();
      },
      None => {
        println!("No message processor channel available");
      }
    }
  }

  fn start_message_processor_thread(&mut self){
    let (sender, receiver) = mpsc::channel::<Message>();
    self.msg_processor_chan = Some(sender.clone());


    let manager = self.uav_manager.clone();
    let sender = self.sender_chan.clone().unwrap();
    thread::spawn(async move || {
      while let Ok(msg) = receiver.recv() {
        let mut message = msg.clone();
        println!("Processing message from UAV {}: {:?}", message.sender_id, message);
        
        // Get message duration based on content size and bandwidth (6Mbps)
        let t_tx = (message.clone().content.unwrap().as_bytes().len() as u64 * 8)/ (6*1000000);

        // Delay message if sender is busy
        if let Some(busy) = manager.get_busy_state(message.sender_id.as_str()) {
          let now = SystemTime::now().duration_since(SystemTime::UNIX_EPOCH).unwrap().as_secs();
          if busy.to > now {
            message.start = busy.to + t_tx;
            //TODO: add also CSMA/CA here
          }
        }

        message.end = Some(message.start + t_tx); 

        // Simulate packet loss based on distance
        if Self::apply_length_filter(manager.clone(), &mut message) {
          continue;
        }

        // Simulate packet loss based on receiver busy state
        if let Some(receiver_id) = message.receiver_id {
          let receiver_busy = manager.get_busy_state(receiver_id.as_str());
          match receiver_busy {
            Some(busy) => {
              if busy.to > message.start || busy.from < message.end.unwrap() {
                println!("Receiver UAV {} is busy during the message transmission. Message from UAV {} to UAV {} lost.", receiver_id, message.sender_id, receiver_id);
                manager.update_busy_state(receiver_id.as_str(), BusyState {
                  from: message.start,
                  to: if busy.to > message.end.unwrap() { busy.to } else { message.end.unwrap()}, 
                  overlapped: true,
                });
                continue;
              } else {
                manager.update_busy_state(receiver_id.as_str(), BusyState { 
                  from: message.start, 
                  to: message.end.unwrap(), 
                  overlapped: false }
                );
              }
            },
            None => {
              manager.update_busy_state(receiver_id.as_str(), BusyState { 
                from: message.start, 
                to: message.end.unwrap(), 
                overlapped: false }
              );
            }
          }
        }

        sender.send((msg, Duration::from_micros(t_tx))).await.unwrap();
      }
    });
  }

  // Simulate packet loss based on distance between sender and receiver
  fn apply_length_filter(manager: UAVManager, msg: &mut Message) -> bool {
    let sender_coords = manager.get_coords(msg.sender_id.as_str()).unwrap();
    let receiver_coords = manager.get_coords(msg.receiver_id.clone().unwrap().as_str()).unwrap();

    let length = ((receiver_coords.x - sender_coords.x).powi(2) + 
      (receiver_coords.y - sender_coords.y).powi(2) + 
      (receiver_coords.z - sender_coords.z).powi(2)
    ).sqrt();

    let loss_packet_prob = 5.335 * (10.0 as f64).powf(-7.0) * length.powi(2) + 3.395 * (10.0 as f64).powf(-5.0) * length;
    let dice = rand::random::<f64>();
    println!("Loss probability: {}, Dice: {}", loss_packet_prob, dice);
    if dice < loss_packet_prob {
      println!("Message from UAV {} to UAV {} lost due to distance {} and loss proabbility {}", msg.sender_id, msg.receiver_id.clone().unwrap(), length, loss_packet_prob);
      return false;
    }

    return true 
  }

  fn start_sender_thread(&mut self)  {
    let (sender, mut receiver) = tokio::sync::mpsc::channel::<(Message, Duration)>(300);
    self.sender_chan = Some(sender.clone());

    let manager = self.uav_manager.clone();
    let socket = UdpSocket::bind("0.0.0.0:0").unwrap();

    tokio::spawn(async move {
      let mut msq_queue: DelayQueue<Message> = DelayQueue::new();
      
      loop {
        tokio::select! {
          Some(value) = msq_queue.next() => {
            let msg = value.into_inner();
            let Some(receiver_id) = msg.clone().receiver_id else {continue;};

            println!("Sending message to UAV {:?}: {:?}", receiver_id, msg);

            let busy_state = manager.get_busy_state(receiver_id.as_str());
            let receiver_addr = manager.get_addr(receiver_id.as_str());

            // socket.send_to(msg.content.unwrap().as_bytes(), addr).unwrap();

            if let (Some(busy), Some(addr)) = (busy_state, receiver_addr) {
              if busy.overlapped {
                println!("Message to UAV {} lost due to collision.", receiver_id);
                continue;
              }

              if let Some(content) = &msg.content {
                socket.send_to(content.as_bytes(), addr).unwrap();
              }

            }
          },

          
          Some((msg, delay)) = receiver.recv() => {
            msq_queue.insert(msg, delay);
          }
        }
      }
    });
  }

  fn start_coords_thread(&mut self) {
    let (sender, receiver) = mpsc::channel::<Message>();
    self.coords_chan = Some(sender.clone());

    let manager = self.uav_manager.clone();
    thread::spawn(move || {
      while let Ok(msg) = receiver.recv() {
        let sender_id = msg.sender_id.as_str();
        println!("Updating UAV {} coords to: {:?}", sender_id, msg.coords);
        manager.update_coords(sender_id, msg.coords.unwrap());
      }
    });
  }
}