use std::net::{SocketAddr, Ipv4Addr, SocketAddrV4};
use std::time::Duration;

use network_module::NetworkSimulator;
use network_module::models::Position;

fn create_addr(port: u16) -> SocketAddr {
    SocketAddr::V4(SocketAddrV4::new(Ipv4Addr::new(127, 0, 0, 1), port))
}

#[test]
fn test_simulator_new() {
    let sim = NetworkSimulator::new("0");
    assert!(sim.get_uavs().is_empty());
    assert!(sim.get_delayed_msgs().is_empty());
    assert!(sim.get_sended_msgs().is_empty());
    assert!(sim.get_uavs_sending().is_empty());
}

#[test]
fn test_update_uav_info() {
    let mut sim = NetworkSimulator::new("0");
    let pos = Position { x: 1.0, y: 2.0, z: 3.0 };
    let addr = create_addr(1000);
    
    sim.update_uav_info("UAV1".to_string(), pos.clone(), addr);
    
    let uav = sim.get_uav("UAV1").unwrap();
    assert_eq!(uav.position, pos);
    assert_eq!(uav.addr, addr);
}

#[test]
fn test_enqueue_message_sender_not_found() {
    let mut sim = NetworkSimulator::new("0");
    sim.enqueue_message("SENDER".to_string(), "TARGET".to_string(), vec![1, 2, 3]);
    
    assert!(sim.get_delayed_msgs().is_empty());
    assert!(sim.get_sended_msgs().is_empty());
}

#[test]
fn test_enqueue_message_target_not_found_but_sender_found() {
    let mut sim = NetworkSimulator::new("0");
    let addr = create_addr(1000);
    sim.update_uav_info("SENDER".to_string(), Position::default(), addr);
    
    let original_busy = sim.get_uav("SENDER").unwrap().busy_until;
    
    // Target doesn't exist, so the message is discarded and sender busy_until is unchanged
    sim.enqueue_message("SENDER".to_string(), "TARGET".to_string(), vec![1, 2, 3]);
    
    assert!(sim.get_delayed_msgs().is_empty());
    assert!(sim.get_sended_msgs().is_empty());
    
    let sender = sim.get_uav("SENDER").unwrap();
    assert_eq!(sender.busy_until, original_busy);
}

#[test]
fn test_enqueue_message_sender_busy() {
    let mut sim = NetworkSimulator::new("0");
    let addr1 = create_addr(1000);
    let addr2 = create_addr(1001);
    
    sim.update_uav_info("UAV1".to_string(), Position::default(), addr1);
    sim.update_uav_info("UAV2".to_string(), Position::default(), addr2);

    // Initial message to make UAV1 busy and send something
    sim.enqueue_message("UAV1".to_string(), "UAV2".to_string(), vec![0; 1000]);
    let first_len = sim.get_sended_msgs().get("UAV2").unwrap().len();
    assert_eq!(first_len, 1);
    assert!(sim.get_delayed_msgs().is_empty());

    // Second message immediately, should be delayed
    sim.enqueue_message("UAV1".to_string(), "UAV2".to_string(), vec![0; 10]);
    assert_eq!(sim.get_sended_msgs().get("UAV2").unwrap().len(), 1); 
    assert_eq!(sim.get_delayed_msgs().len(), 1);
}

#[test]
fn test_enqueue_message_success() {
    let mut sim = NetworkSimulator::new("0");
    let addr1 = create_addr(1000);
    let addr2 = create_addr(1001);
    
    // Very close positions so packet drop probability is effectively 0
    sim.update_uav_info("UAV1".to_string(), Position { x: 0.0, y: 0.0, z: 0.0 }, addr1);
    sim.update_uav_info("UAV2".to_string(), Position { x: 0.0000001, y: 0.0, z: 0.0 }, addr2);
    
    sim.enqueue_message("UAV1".to_string(), "UAV2".to_string(), vec![1, 2, 3]);
    
    assert_eq!(sim.get_sended_msgs().get("UAV2").unwrap().len(), 1);
    let msg = &sim.get_sended_msgs().get("UAV2").unwrap()[0];
    assert_eq!(msg.sender_id, "UAV1");
    assert_eq!(msg.target_id, "UAV2");
    assert_eq!(msg.tx, 4); // 3 * 8 / 6 = 4 micros
}

#[test]
fn test_send_messages() {
    let mut sim = NetworkSimulator::new("0");
    let addr1 = create_addr(1000);
    let addr2 = create_addr(1001);
    
    sim.update_uav_info("UAV1".to_string(), Position { x: 0.0, y: 0.0, z: 0.0 }, addr1);
    sim.update_uav_info("UAV2".to_string(), Position { x: 0.0, y: 0.0, z: 0.0 }, addr2);
    
    // Queue some delayed messages explicitly for test logic by making it busy first
    sim.enqueue_message("UAV1".to_string(), "UAV2".to_string(), vec![0; 1000]); // Tx is 1333 micros
    sim.enqueue_message("UAV1".to_string(), "UAV2".to_string(), vec![1; 10]); // This one will be delayed
    
    assert_eq!(sim.get_delayed_msgs().len(), 1);
    
    // Wait for busy time to pass so the delayed message can be processed successfully upon send_messages
    std::thread::sleep(Duration::from_millis(2));
    
    // Send it
    sim.send_messages();
    
    // The delayed msg should be re-enqueued, which will put it into sended_msgs since it's no longer busy!
    // However, send_messages loops through and processes delayed ones. They will be placed into the NEW sended messages buffer for the NEXT send loop
    assert_eq!(sim.get_sended_msgs().get("UAV2").unwrap().len(), 1);
    assert!(sim.get_delayed_msgs().is_empty());
}

#[test]
fn test_is_near_senders_colliding_chunks() {
    let mut sim = NetworkSimulator::new("0");
    let addr1 = create_addr(1000);
    let addr2 = create_addr(1001);
    let addr3 = create_addr(1002);
    
    // UAV1 and UAV3 are in the same chunk. UAV2 is target.
    sim.update_uav_info("UAV1".to_string(), Position { x: 0.0, y: 0.0, z: 0.0 }, addr1);
    sim.update_uav_info("UAV2".to_string(), Position { x: 0.0, y: 0.0, z: 0.0 }, addr2);
    sim.update_uav_info("UAV3".to_string(), Position { x: 0.001, y: 0.0, z: 0.0 }, addr3);

    // Make UAV1 send a big message to UAV2 so it becomes busy
    sim.enqueue_message("UAV1".to_string(), "UAV2".to_string(), vec![0; 1000]);

    // Now UAV3 tries to send to UAV2
    // Even though UAV3 is not personally busy, another UAV in its chunk is transmitting.
    // is_near_senders should return true, and the message should be DELAYED.
    sim.enqueue_message("UAV3".to_string(), "UAV2".to_string(), vec![0; 10]);

    // We should have 1 queued delayed message (from UAV3)
    assert_eq!(sim.get_delayed_msgs().len(), 1);
    let delayed = &sim.get_delayed_msgs()[0];
    assert_eq!(delayed.sender_id, "UAV3");
}

#[test]
fn test_update_uav_info_with_first_attempt_should_add_new_uav() {
    let mut sim = NetworkSimulator::new("0");
    let pos = Position::default();
    let addr: SocketAddr = "127.0.0.1:4005".parse().unwrap();
    let uav_id = String::from("1");

    sim.update_uav_info(uav_id.clone(), pos.clone(), addr);
    let new_uav = sim.get_uav(&uav_id);

    assert!(new_uav.is_some(), "uav not created");

    let new_uav = new_uav.unwrap();
    assert_eq!(new_uav.addr, addr);
    assert_eq!(new_uav.position, pos);
}
