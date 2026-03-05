
use std::{cmp::Ordering, net::SocketAddr, time::Instant};

use serde::Deserialize;

#[derive(Clone, Debug, Default, Deserialize)]
pub struct Position {
  pub x: f64,
  pub y: f64,
  pub z: f64,
}

impl PartialEq for Position {
  fn eq(&self, other: &Self) -> bool {
      self.x == other.x && 
      self.y == other.y &&
      self.z == other.z
  }
}

#[derive(Clone, Debug)]
pub struct UAV {
  pub position: Position,
  pub busy_until: Instant,
  pub addr: SocketAddr,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Message {
  pub sender_id: String,
  pub target_id: String,
  pub target_addr: Option<SocketAddr>,
  pub payload: Vec<u8>,
  pub from: Instant,
  pub to: Instant,
  pub tx: u64, // Transmission time in microseconds
  pub overlapped: bool,
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