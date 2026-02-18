use std::net::UdpSocket;

use anyhow::Result;

use crate::network_manager::network_manager::NetworkManager;


mod network_manager;

const MAX_BUFFER_OPERATING_SYS:usize = 65507;

fn main() {
    let socket = UdpSocket::bind("0.0.0.0:3000").expect("couldn't bind to address");

    let local_addr = socket.local_addr().expect("couldn't get local address");
    println!("Listening on {}", local_addr);

    let manager = NetworkManager::new();
    
    loop {
        let msg = match get_msg(&socket) {
            Ok(msg) => msg,
            Err(_) => {
                println!("Cannot receive json message");
                continue;
            }
        };

        manager.push(msg);
    }
}

fn get_msg(socket: &UdpSocket) -> Result<serde_json::Value>{
    let mut buf  = [0; MAX_BUFFER_OPERATING_SYS];

    let (number_of_bytes, _) = socket.recv_from(&mut buf)?;
    let filled_buff = &mut buf[..number_of_bytes];

    let parsed = serde_json::from_slice(filled_buff)?;

    Ok(parsed)
}

