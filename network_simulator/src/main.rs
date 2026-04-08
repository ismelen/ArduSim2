//! # Network Simulator Broker
//!
//! UDP broker that listens for datagrams from UAV clients and simulates
//! a broadcast wireless network. It manages carrier sensing, distance-based
//! packet loss, and collisions.
//!
//! ## Usage
//!
//! ```bash
//! ./network_simulator <listen_port> <send_port> [buffer_size]
//! ```
//!
//! ## Message Formats (JSON)
//!
//! Telemetry update:
//! ```json
//! {
//!   "topic": "telemetry",
//!   "payload": {
//!     "uav_id": "drone_0",
//!     "payload": {
//!       "nr_gps_online": 10,
//!       "position": {"heading": 0.0, "alt": 10.0, "relative_alt": 10.0, "lon": 0.0, "lat": 0.0},
//!       "type": "MAV_TYPE_QUADROTOR",
//!       "battery": 100,
//!       "version": "1.0",
//!       "time_boot_ms": 1000,
//!       "speed": {"vx": 0.0, "vy": 0.0, "vz": 0.0},
//!       "status": "OK",
//!       "flight_mode": "GUIDED"
//!     }
//!   }
//! }
//! ```
//!
//! Broadcast message:
//! ```json
//! {
//!   "topic": "broadcast",
//!   "payload": {
//!     "uav_id": "drone_0",
//!     "payload": "Hello"
//!   }
//! }
//! ```

use std::{
    net::UdpSocket, sync::Arc, time::{Duration, Instant}
};

use network_simulator::config::{MAX_DATAGRAM_SIZE, SWITCH_INTERVAL};
use network_simulator::NetworkSimulator;
use serde::Deserialize;

use clap::Parser;
use network_simulator::logger::{LoggerFactory, LogMode};

/// Command line arguments for the simulator.
#[derive(Parser, Debug)]
#[command(author, version, about = "Network Simulator Broker", long_about = None)]
struct Args {
    /// Port to listen for incoming datagrams from UAVs
    #[arg(long, default_value_t = 3000)]
    port: u16,

    /// Buffer size for receiving datagrams
    #[arg(long, default_value_t = network_simulator::config::DEFAULT_BUFFER_SIZE)]
    buffer_size: usize,

    /// Logging mode: debug, debug-important, prod
    #[arg(long, default_value = "debug")]
    mode: String,

    /// IP address for the remote logger (required if in prod mode)
    #[arg(long)]
    logger_ip: Option<String>,

    /// Port for the remote logger (required if in prod mode)
    #[arg(long)]
    logger_port: Option<u16>,
}

/// Incoming UDP datagram structure from a client or UAV.
#[derive(Deserialize, Debug)]
#[serde(tag = "topic", content = "payload", rename_all = "lowercase")]
pub enum UdpMessage {
    /// Request to subscribe to updates.
    Subscribe {
        topic: String,
    },
    /// Request to broadcast a generic payload.
    Broadcast {
        uav_id: String,
        payload: serde_json::Value,
    },
    /// A telemetry update from a UAV.
    Telemetry{
        payload: network_simulator::models::TelemetryData,
        uav_id: String,
    },
}

/// Wrapper to attach the socket address to the parsed message.
struct ReceivedMessage {
    msg: UdpMessage,
    addr: std::net::SocketAddr,
}

/// Main entry point for the network simulator broker.
fn main() {
    let args = Args::parse();

    let mode = match args.mode.as_str() {
        "debug" => LogMode::Debug,
        "debug-important" => LogMode::DebugImportant,
        "prod" => {
            let ip = args.logger_ip.expect("--logger-ip is required when mode is prod");
            let port = args.logger_port.expect("--logger-port is required when mode is prod");
            LogMode::Prod { ip, port }
        }
        _ => panic!("Unknown mode: {}. Valid modes: debug, debug-important, prod", args.mode),
    };

    let logger = std::sync::Arc::new(LoggerFactory::create(mode));

    let bind_addr = format!("0.0.0.0:{}", args.port);
    let socket = Arc::new(UdpSocket::bind(&bind_addr).expect(&format!("couldn't bind to {}", bind_addr)));
    socket
        .set_read_timeout(Some(Duration::from_micros(100)))
        .expect("couldn't set read timeout");

    println!("Network simulator listening on {}", bind_addr);
    println!(
        "Sending on port {}, buffer size: {} bytes",
        args.port, args.buffer_size
    );

    let mut sim = NetworkSimulator::new(socket.clone(), args.buffer_size, logger);
    let mut last_flush = Instant::now();

    let running = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(true));
    {
        let r = running.clone();
        ctrlc::set_handler(move || {
            r.store(false, std::sync::atomic::Ordering::Relaxed);
        }).expect("Error setting Ctrl-C handler");
    }

    while running.load(std::sync::atomic::Ordering::Relaxed) {
        if last_flush.elapsed() >= SWITCH_INTERVAL {
            last_flush = Instant::now();
            sim.send_messages();
        }

        if let Some(received) = receive_datagram(&socket) {
            let start = Instant::now();

            match received.msg {
                UdpMessage::Subscribe { topic } => {
                    sim.subscribe(received.addr, topic);
                }
                UdpMessage::Broadcast { uav_id: sender_id, payload } => {
                    sim.enqueue_broadcast(sender_id, serde_json::to_string(&payload).unwrap(), 0);
                }
                UdpMessage::Telemetry{ uav_id: sender_id, payload: telemetry_data } => {
                    sim.update_uav_info(telemetry_data, sender_id, received.addr);
                }
            }

            last_flush -= start.elapsed();
        }
    }

    sim.print_stats();
}

/// Try to read and deserialize a single UDP datagram from the socket.
fn receive_datagram(socket: &UdpSocket) -> Option<ReceivedMessage> {
    let mut buf = [0u8; MAX_DATAGRAM_SIZE];

    if let Ok((len, addr)) = socket.recv_from(&mut buf) {
        if let Ok(msg) = serde_json::from_slice::<UdpMessage>(&buf[..len]) {
            return Some(ReceivedMessage { msg, addr });
        }
    }

    None
}


