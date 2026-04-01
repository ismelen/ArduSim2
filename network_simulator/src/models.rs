//! # Data Models
//!
//! Contains the core structures used by the network simulator, including
//! positions, UAV representations, and network messages.

use std::{cmp::Ordering, net::SocketAddr, sync::Arc, time::Instant};

use serde::{Deserialize, Serialize};

use crate::METERS_PER_DEGREE;

/// 3D position (geographic coordinates or UTM-like).
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Position {
    /// X coordinate (e.g., longitude degrees or meters).
    pub x: f64,
    /// Y coordinate (e.g., latitude degrees or meters).
    pub y: f64,
    /// Z coordinate (e.g., altitude in meters).
    pub z: f64,
}

impl PartialEq for Position {
    fn eq(&self, other: &Self) -> bool {
        self.x == other.x && self.y == other.y && self.z == other.z
    }
}

/// A registered UAV in the simulator.
#[derive(Clone, Debug)]
pub struct UAV {
    /// Unique ID of the UAV.
    pub id: String,

    /// Current 3D position.
    pub position: Position,

    /// Instant until which this UAV is busy transmitting.
    pub busy_until: Instant,

    /// UDP address to deliver messages to this UAV.
    pub addr: SocketAddr,

    /// Bytes currently occupying the receiving buffer.
    pub buffer_used: usize,

    /// Spatial chunk key this UAV currently belongs to.
    pub chunk_key: (i64, i64, i64),
}

/// A broadcast message being transmitted through the simulated network.
///
/// Ordered by transmission start time (`from`) to detect temporal overlaps.
#[derive(Clone, Debug)]
pub struct Message {
    /// ID of the sending UAV.
    pub sender_id: String,

    /// Shared payload bytes (avoids cloning per receiver).
    pub payload: Arc<Vec<u8>>,

    /// Transmission start instant.
    pub from: Instant,

    /// Transmission end instant.
    pub to: Instant,

    /// Transmission time in nanoseconds.
    pub tx_ns: u64,

    /// Whether this message overlaps temporally with another.
    pub overlapped: bool,

    /// Number of times this message has been delayed and retried.
    pub retries: u32,

    /// Resolved target address (populated when routing to a specific receiver).
    pub target_addr: Option<SocketAddr>,
}

impl PartialEq for Message {
    fn eq(&self, other: &Self) -> bool {
        self.sender_id == other.sender_id && self.from == other.from
    }
}

impl Eq for Message {}

impl Ord for Message {
    fn cmp(&self, other: &Self) -> Ordering {
        self.from.cmp(&other.from)
    }
}

impl PartialOrd for Message {
    fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
        Some(self.from.cmp(&other.from))
    }
}

impl Default for Message {
    fn default() -> Self {
        Self {
            sender_id: String::new(),
            payload: Arc::new(Vec::new()),
            from: Instant::now(),
            to: Instant::now(),
            tx_ns: 0,
            overlapped: false,
            retries: 0,
            target_addr: None,
        }
    }
}

/// A telemetry position from ArduSim (raw GPS degrees).
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct TelemetryPosition {
    pub heading: f64,
    pub alt: f64,
    pub relative_alt: f64,
    pub lon: f64,
    pub lat: f64,
}

impl TelemetryPosition {
    /// Converts the geographic coordinates to meters relative to some origin logic.
    /// Following the Java logic:
    /// Longitude (degrees) * 111.12 km * 1000 m = X offset
    /// Latitude (degrees) * 111.12 km * 1000 m = Y offset
    pub fn to_sim_position(&self) -> Position {
        Position {
            x: self.lon * METERS_PER_DEGREE,
            y: self.lat * METERS_PER_DEGREE,
            z: self.alt,
        }
    }
}

/// A telemetry speed update from ArduSim.
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct TelemetrySpeed {
    pub vx: f64,
    pub vy: f64,
    pub vz: f64,
}

/// A combined telemetry update received via UDP.
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct TelemetryData {
    pub nr_gps_online: i32,
    pub position: TelemetryPosition,
    #[serde(rename = "type")]
    pub uav_type: String,
    pub battery: i32,
    pub version: String,
    pub time_boot_ms: u64,
    pub speed: TelemetrySpeed,
    pub status: String,
    pub flight_mode: String,
}

