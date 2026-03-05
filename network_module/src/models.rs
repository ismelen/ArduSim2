
use std::{cmp::Ordering, net::SocketAddr, time::Instant};

use serde::Deserialize;

/// Represents a 3D position in geographic coordinate space.
///
/// Coordinates use decimal degrees (latitude/longitude) for `x` and `y`,
/// and a similar scale for altitude `z`. These are used to calculate
/// inter-UAV distances for signal propagation and collision detection.
#[derive(Clone, Debug, Default, Deserialize)]
pub struct Position {
  /// Latitude-like coordinate (decimal degrees).
  pub x: f64,
  /// Longitude-like coordinate (decimal degrees).
  pub y: f64,
  /// Altitude coordinate.
  pub z: f64,
}

impl PartialEq for Position {
  fn eq(&self, other: &Self) -> bool {
      self.x == other.x && 
      self.y == other.y &&
      self.z == other.z
  }
}

/// Represents an Unmanned Aerial Vehicle (UAV) registered in the network simulator.
///
/// Each UAV is identified by a string ID and tracks its current position,
/// the UDP address where it listens for messages, and a transmission busy flag.
#[derive(Clone, Debug)]
pub struct UAV {
  /// Current 3D position of the UAV.
  pub position: Position,
  /// The time instant until which this UAV is busy transmitting.
  /// Messages arriving before this instant will be delayed.
  pub busy_until: Instant,
  /// The UDP socket address of this UAV for message delivery.
  pub addr: SocketAddr,
}

/// Represents a message being transmitted between two UAVs through the network simulator.
///
/// Messages are ordered by their transmission start time ([`from`](Message::from))
/// to detect signal overlaps (collisions) at the receiver.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Message {
  /// ID of the UAV sending this message.
  pub sender_id: String,
  /// ID of the target UAV.
  pub target_id: String,
  /// Resolved UDP address of the target (populated when enqueued).
  pub target_addr: Option<SocketAddr>,
  /// Raw payload bytes to be delivered.
  pub payload: Vec<u8>,
  /// Transmission start time.
  pub from: Instant,
  /// Transmission end time (`from` + transmission duration).
  pub to: Instant,
  /// Transmission time in microseconds, calculated as `payload_bytes * 8 / 6`.
  pub tx: u64,
  /// Whether this message's transmission window overlaps with another message.
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