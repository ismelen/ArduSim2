use std::{net::{SocketAddr, UdpSocket}, time::{Duration, Instant}};

use network_module::{NetworkSimulator, models::Position};
use serde::Deserialize;


const MAX_BUFFER_OPERATIN_SYS: usize = 65507;
const SWITCH_INTERVAL: Duration = Duration::from_millis(1);

#[derive(Deserialize)]
struct UdpMessage {
	sender_id: String,
	target_id: Option<String>,
	position: Option<Position>,
	payload: Option<Vec<u8>>,
	addr: Option<SocketAddr>,
}

fn main() {
	let socket = UdpSocket::bind("0.0.0.0:3000").expect("couldn't bidn to address");
	socket.set_read_timeout(Some(Duration::from_micros(100))).expect("couldn't set read timeout");
	// socket.set_nonblocking(true).expect("couldn't set non-blocking");

	let mut ns = NetworkSimulator::new("3001");
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
