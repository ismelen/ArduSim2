//! # Network Simulator
//!
//! UDP broker that simulates a WiFi ad-hoc broadcast network between UAVs.
//!
//! Applies logic for:
//! - **Carrier sensing (CSMA)**: delays transmission if nearby UAVs are sending.
//! - **Collision detection**: overlapping transmissions at the receiver are dropped.
//! - **Distance-based packet loss**: probabilistic model based on WiFi 5GHz.
//! - **Buffer overflow**: receivers have a limited buffer; excess messages are dropped.
//! - **Broadcast routing**: messages from individuals to all registered neighborhood UAVs.
//!
//! ## Spatial chunk indexing
//!
//! UAVs are stored in a spatial grid (chunks of ~0.5km). Broadcasting uses
//! a 5x5x5 neighborhood lookup to find receivers within range, eliminating
//! the need for an O(N) distance check against every UAV.

pub mod config;
pub mod logger;
pub mod models;
pub mod dispatcher;
pub mod simulator;
pub mod spatial;
pub mod telemetry;

pub use config::*;
pub use simulator::NetworkSimulator;
