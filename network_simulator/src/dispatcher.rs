use std::cmp;
use std::collections::HashMap;
use std::net::UdpSocket;
use std::sync::mpsc;
use std::sync::Arc;
use std::thread;
use serde_json::Map;
use serde_json::json;

use crate::logger::Logger;
use crate::models::Message;

/// Handles moving logically validated transmissions off the main simulator mapping threads synchronously natively.
pub struct UdpDispatcher {
    /// Hand-off path passing outbound messages toward the external persistence routine.
    send_tx: mpsc::Sender<HashMap<String, Vec<Message>>>,
}

impl UdpDispatcher {
    pub fn new(socket: Arc<UdpSocket>, logger: Arc<Logger>) -> Self {
        let (send_tx, send_rx) = mpsc::channel::<HashMap<String, Vec<Message>>>();

        thread::spawn(move || {
            Self::sender_loop(send_rx, socket, logger);
        });

        Self { send_tx }
    }

    /// Appends the outgoing queue logically pushing items for background processing delivery via threads.
    pub fn send_messages(&self, pending_msgs: HashMap<String, Vec<Message>>) {
        let _ = self.send_tx.send(pending_msgs);
    }

    /// Background system running independent tracking of deliveries parsing out error sets natively.
    fn sender_loop(
        rx: mpsc::Receiver<HashMap<String, Vec<Message>>>,
        socket: Arc<UdpSocket>,
        logger: Arc<Logger>,
    ) {
        while let Ok(pending) = rx.recv() {
            for (receiver_id, msgs) in &pending {
                if msgs.is_empty() {
                    continue;
                }

                let mut send_next = true;
                let mut current = &msgs[0];
                let mut max_end = current.to;

                for i in 1..msgs.len() {
                    let next = &msgs[i];
                    let overlapped = next.from <= max_end;

                    if !overlapped && send_next {
                        Self::send_udp(&socket, current, &logger);
                    } else if overlapped {
                        logger.msg_overlapped(receiver_id);
                    }

                    current = next;
                    max_end = cmp::max(current.to, max_end);
                    send_next = !overlapped;
                }

                if send_next {
                    Self::send_udp(&socket, current, &logger);
                }
            }
        }
    }

    /// Securely isolates outbound requests parsing basic delivery outcomes tracking elements securely.
    fn send_udp(socket: &UdpSocket, msg: &Message, logger: &Logger) {
        if let Some(addr) = msg.target_addr {
            let mut map = Map::new();
            let json_payload = serde_json::from_str(&msg.payload.to_string()).unwrap();
            map.insert("uav_id".to_string(), json!(msg.sender_id));
            map.insert("payload".to_string(), json_payload);

            let json = serde_json::to_string(&map).unwrap();
            match socket.send_to(json.as_bytes(), addr) {
                Ok(_) => logger.msg_sent_ok(&format!("{}", addr)),
                Err(e) => logger.msg_sent_err(&format!("{}", addr), &e.to_string()),
            }
        }
    }
}
