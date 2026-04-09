use std::collections::HashSet;
use std::net::{SocketAddr, UdpSocket};
use std::sync::Arc;

use crate::logger::Logger;
use crate::models::TelemetryData;

/// Manages external clients configured via sockets looking to receive native simulation object offsets and messages.
pub struct TelemetryManager {
    /// Subscribed clients mapped by topic.
    subscriptions: std::collections::HashMap<String, HashSet<SocketAddr>>,
    /// Socket used to broadcast outbound telemetry securely across UDP.
    socket: Arc<UdpSocket>,
    /// Global logger linking metric triggers naturally across native environments.
    logger: Arc<Logger>,
}

impl TelemetryManager {
    pub fn new(socket: Arc<UdpSocket>, logger: Arc<Logger>) -> Self {
        Self {
            subscriptions: std::collections::HashMap::new(),
            socket,
            logger,
        }
    }

    /// Registers a new subscriber for updates on a specific topic.
    pub fn subscribe(&mut self, addr: SocketAddr, topic: String) {
        if self.subscriptions.entry(topic.clone()).or_default().insert(addr) {
            self.logger.topic_susbscribed(&addr, &topic);
        }
    }

    /// Pushes structured JSON streams automatically across registered client target sockets.
    pub fn relay_telemetry(&self, telemetry: &TelemetryData, uav_id: &str) {
        let subscribers = match self.subscriptions.get("telemetry") {
            Some(subs) if !subs.is_empty() => subs,
            _ => return,
        };

        if let Ok(json_payload) = serde_json::to_vec(&serde_json::json!({
            "topic": "telemetry",
            "uav_id": uav_id,
            "payload": telemetry
        })) {
            for sub_addr in subscribers {
                let _ = self.socket.send_to(&json_payload, sub_addr);
            }
            self.logger.telemetry_relayed();
        }
    }

    /// Pushes broadcast messages to clients subscribed to the 'messages' topic.
    pub fn relay_message(&self, payload: &str, sender_id: &str) {
        let subscribers = match self.subscriptions.get("messages") {
            Some(subs) if !subs.is_empty() => subs,
            _ => return,
        };

        // Note: attempting to parse so we can send a nested JSON payload if valid
        let payload_val: serde_json::Value = serde_json::from_str(payload)
            .unwrap_or_else(|_| serde_json::json!(payload));

        if let Ok(json_payload) = serde_json::to_vec(&serde_json::json!({
            "topic": "messages",
            "uav_id": sender_id,
            "payload": payload_val
        })) {
            for sub_addr in subscribers {
                let _ = self.socket.send_to(&json_payload, sub_addr);
            }
        }
    }
}
