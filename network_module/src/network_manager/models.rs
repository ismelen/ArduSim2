use serde::Deserialize;

#[derive(Deserialize, Debug, Clone)]
pub struct Message {
  pub sender_id: String,
  pub receiver_id: Option<String>,
  pub content: Option<String>,
  pub coords: Option<Coords>,
  pub start: u64,
  pub end: Option<u64>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct Coords {
  pub x: f64,
  pub y: f64,
  pub z: f64,
}

#[derive(Debug, Clone)]
pub struct BusyState {
  pub from: u64,
  pub to: u64,
  pub overlapped: bool,
}
