use std::collections::HashSet;
use std::net::{SocketAddr, UdpSocket};
use std::sync::Arc;

use crate::logger::Logger;
use crate::models::TelemetryData;

/// Manages external clients configured via sockets looking to receive native simulation object offsets.
pub struct TelemetryManager {
    /// Subscribed clients for telemetry updates.
    subscribers: HashSet<SocketAddr>,
    /// Socket used to broadcast outbound telemetry securely across UDP.
    socket: Arc<UdpSocket>,
    /// Global logger linking metric triggers naturally across native environments.
    logger: Arc<Logger>,
}

impl TelemetryManager {
    pub fn new(socket: Arc<UdpSocket>, logger: Arc<Logger>) -> Self {
        Self {
            subscribers: HashSet::new(),
            socket,
            logger,
        }
    }

    /// Registers a new subscriber for telemetry updates.
    pub fn subscribe(&mut self, addr: SocketAddr) {
        if self.subscribers.insert(addr) {
            self.logger.telemetry_subscribed(&addr);
        }
    }

    /// Pushes structured JSON streams automatically across registered client target sockets.
    pub fn relay_telemetry(&self, telemetry: &TelemetryData) {
        if self.subscribers.is_empty() {
            return;
        }

        if let Ok(json_payload) = serde_json::to_vec(telemetry) {
            for sub_addr in &self.subscribers {
                let _ = self.socket.send_to(&json_payload, sub_addr);
            }
            self.logger.telemetry_relayed();
        }
    }
}
