//! # Network Module Broker
//!
//! UDP broker that acts as the central hub for the network simulator.
//!
//! Listens for incoming UDP datagrams from UAV clients.
//! Each datagram is either a **position update** (no `target_id`) or a
//! **data message** (with `target_id` and `payload`).
//!
//! ## Usage
//!
//! ```bash
//! ./network_module <listen_port> <send_port>
//! ```
//!
//! - `listen_port`: UDP port to receive datagrams from UAVs (default: `3000`).
//! - `send_port`: UDP port to send messages to UAVs (default: `3001`).
//!
//! The broker runs a tight loop that alternates between:
//! 1. Flushing enqueued messages every [`SWITCH_INTERVAL`].
//! 2. Receiving and processing incoming UDP datagrams.

use std::{env, net::{SocketAddr, UdpSocket}, time::{Duration, Instant}};

use network_module::{NetworkSimulator, models::Position};
use serde::Deserialize;

/// Maximum UDP datagram size supported by the OS.
const MAX_BUFFER_OPERATIN_SYS: usize = 65507;

/// Interval between message flush cycles.
/// Controls how often [`NetworkSimulator::send_messages`] is called.
const SWITCH_INTERVAL: Duration = Duration::from_millis(1);

/// Default UDP port for receiving datagrams from UAVs.
const DEFAULT_LISTEN_PORT: &str = "3000";

/// Default UDP port for sending messages to UAVs.
const DEFAULT_SEND_PORT: &str = "3001";

/// Represents an incoming UDP datagram from a UAV client.
///
/// - If `target_id` is `None`, this is a **position update**.
/// - If `target_id` is `Some`, this is a **data message** with a payload.
#[derive(Deserialize)]
struct UdpMessage {
	/// ID of the sending UAV.
	sender_id: String,
	/// ID of the target UAV (if this is a data message).
	target_id: Option<String>,
	/// Updated position of the sender (if this is a position update).
	position: Option<Position>,
	/// Message payload bytes (if this is a data message).
	payload: Option<Vec<u8>>,
	/// Source address of the UDP datagram (populated after deserialization).
	#[serde(skip)]
	addr: Option<SocketAddr>,
}

fn main() {
	let args: Vec<String> = env::args().collect();
	let listen_port = args.get(1).map(|s| s.as_str()).unwrap_or(DEFAULT_LISTEN_PORT);
	let send_port = args.get(2).map(|s| s.as_str()).unwrap_or(DEFAULT_SEND_PORT);

	let bind_addr = format!("0.0.0.0:{}", listen_port);
	let socket = UdpSocket::bind(&bind_addr).expect(&format!("couldn't bind to {}", bind_addr));
	socket.set_read_timeout(Some(Duration::from_micros(100))).expect("couldn't set read timeout");

	let mut ns = NetworkSimulator::new(send_port);
	let mut last_switch = Instant::now();

	loop {
		if last_switch.elapsed() >= SWITCH_INTERVAL {
			last_switch = Instant::now();
			ns.send_messages();
		}
		
		let msg = get_new_msg(&socket);
		if msg.is_none() { continue; }
		let msg = msg.unwrap();

		let start = Instant::now();
		if msg.target_id.is_none() {
			ns.update_uav_info(msg.sender_id, msg.position.unwrap(), msg.addr.unwrap());
		}else {
			ns.enqueue_message(msg.sender_id, msg.target_id.unwrap(), msg.payload.unwrap());
		}

		last_switch -= start.elapsed();
	}
}

/// Attempts to read and deserialize a single UDP datagram from the socket.
///
/// Returns `None` if no datagram is available (timeout) or if deserialization fails.
fn get_new_msg(socket: &UdpSocket) -> Option<UdpMessage>{ 
	let mut buf = [0; MAX_BUFFER_OPERATIN_SYS];

	if let Ok((len, addr)) = socket.recv_from(&mut buf) {
		let buf = &mut buf[..len];
		if let Ok(mut msg) = serde_json::from_slice::<UdpMessage>(buf) {
			msg.addr = Some(addr);
			return Some(msg)
		}
	}

	None
}
