use serde::{Deserialize, Serialize};

#[derive(Deserialize, Debug, Clone, Serialize)]
pub struct Message {
  pub sender_id: String,
  pub receiver_id: Option<String>,
  pub content: Option<String>,
  pub coords: Option<Coords>,
  pub start: u128,
  pub end: Option<u128>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Coords {
  pub x: f64,
  pub y: f64,
  pub z: f64,
}

impl PartialEq for Coords {
  fn eq(&self, other: &Self) -> bool {
    self.x == other.x && self.y == other.y && self.z == other.z
  }
}

#[derive(Debug, Clone)]
pub struct BusyState {
  pub from: u128,
  pub to: u128,
  pub overlapped: bool,
}
