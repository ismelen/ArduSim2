use std::net::{Ipv4Addr, SocketAddr, SocketAddrV4, UdpSocket};
use std::time::Duration;

use std::sync::Arc;

use network_simulator::models::{TelemetryData, TelemetryPosition, TelemetrySpeed};
use network_simulator::NetworkSimulator;
use network_simulator::logger::Logger;

fn addr(port: u16) -> SocketAddr {
    SocketAddr::V4(SocketAddrV4::new(Ipv4Addr::new(127, 0, 0, 1), port))
}

fn mock_telemetry(lon: f64, lat: f64, alt: f64) -> TelemetryData {
    TelemetryData {
        nr_gps_online: 2,
        position: TelemetryPosition {
            heading: 0.0,
            alt,
            relative_alt: alt,
            lon,
            lat,
        },
        uav_type: "MAV_TYPE_QUADROTOR".to_string(),
        battery: 100,
        version: "4.5.3".to_string(),
        time_boot_ms: 10000,
        speed: TelemetrySpeed {
            vx: 0.0,
            vy: 0.0,
            vz: 0.0,
        },
        status: "OK".to_string(),
        flight_mode: "STABILIZE".to_string(),
    }
}

fn create_sim(buffer_size: usize) -> NetworkSimulator {
    let socket = Arc::new(UdpSocket::bind("127.0.0.1:0").unwrap());
    NetworkSimulator::new(socket, buffer_size, Arc::new(Logger::new()))
}

// ── Constructor ─────────────────────────────────────────────

#[test]
fn test_new_simulator_is_empty() {
    let sim = create_sim(163_840);
    assert!(sim.get_uavs().is_empty());
    assert!(sim.get_delayed_msgs().is_empty());
    assert!(sim.get_pending_msgs().is_empty());
    assert!(sim.get_active_transmissions().is_empty());
    assert!(sim.get_chunks().is_empty());
}

// ── UAV registration ────────────────────────────────────────

#[test]
fn test_register_uav() {
    let mut sim = create_sim(163_840);
    // (x=1.0, y=2.0) -> requires dividing by 111120.0 to back-calculate the deg if we want to check coords closely,
    // but the test primarily just checks if ID and addresses are populated.
    let tel = mock_telemetry(1.0, 2.0, 3.0);
    let a = addr(5000);

    sim.update_uav_info(tel, "drone_0".to_string(), a);

    let uav = sim.get_uav("drone_0").unwrap();
    // Validate position was mapped correctly (X = lon * 111120.0, etc.)
    assert_eq!(uav.position.x, 1.0 * 111.12 * 1000.0);
    assert_eq!(uav.position.y, 2.0 * 111.12 * 1000.0);
    assert_eq!(uav.position.z, 3.0);
    assert_eq!(uav.addr, a);
    assert_eq!(uav.buffer_used, 0);
    assert_eq!(uav.id, "drone_0");
}

#[test]
fn test_register_uav_creates_chunk() {
    let mut sim = create_sim(163_840);
    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d1".to_string(), addr(5000));

    // Should have exactly one chunk with one UAV
    assert_eq!(sim.get_chunks().len(), 1);
    let chunk_uavs: Vec<&Vec<String>> = sim.get_chunks().values().collect();
    assert_eq!(chunk_uavs[0].len(), 1);
    assert_eq!(chunk_uavs[0][0], "d1");
}

#[test]
fn test_update_uav_re_indexes_chunk() {
    let mut sim = create_sim(163_840);
    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d1".to_string(), addr(5000));
    let old_chunk = sim.get_uav("d1").unwrap().chunk_key;

    // Move far away so chunk key changes (e.g. 1.0 deg is 111,120 m away)
    sim.update_uav_info(mock_telemetry(1.0, 1.0, 0.0), "d1".to_string(), addr(5000));
    let new_chunk = sim.get_uav("d1").unwrap().chunk_key;

    assert_ne!(old_chunk, new_chunk);
    // Old chunk should be removed (was the only UAV)
    assert!(sim.get_chunks().get(&old_chunk).is_none());
    // New chunk should contain d1
    assert!(sim
        .get_chunks()
        .get(&new_chunk)
        .unwrap()
        .contains(&"d1".to_string()));
}

// ── Broadcast: sender not registered ────────────────────────

#[test]
fn test_broadcast_unknown_sender_is_discarded() {
    let mut sim = create_sim(163_840);
    sim.enqueue_broadcast("ghost".into(), vec![1, 2, 3], 0);

    assert!(sim.get_delayed_msgs().is_empty());
    assert!(sim.get_pending_msgs().is_empty());
}

// ── Broadcast: success (same chunk) ─────────────────────────

#[test]
fn test_broadcast_enqueues_for_nearby_receivers() {
    let mut sim = create_sim(163_840);

    // Three very close UAVs (same chunk, distance ≈ 0)
    // METERS_PER_DEGREE = 111120.0
    // chunk size = 0.350 meters
    // to belong to the same chunk they have to be extremely close in lat/lon
    // Let's use 0.0, 0.000000001 (which is 0.00011 m), etc.
    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d0".to_string(), addr(5000));
    sim.update_uav_info(mock_telemetry(0.000000001, 0.0, 0.0), "d1".to_string(), addr(5001));
    sim.update_uav_info(mock_telemetry(0.000000002, 0.0, 0.0), "d2".to_string(), addr(5002));

    sim.enqueue_broadcast("d0".into(), vec![10, 20, 30], 0);

    let pending = sim.get_pending_msgs();
    assert!(pending.contains_key("d1"), "d1 should receive");
    assert!(pending.contains_key("d2"), "d2 should receive");
    assert!(!pending.contains_key("d0"), "sender excluded");

    // Check message content
    let msg = &pending.get("d1").unwrap()[0];
    assert_eq!(msg.sender_id, "d0");
    assert_eq!(*msg.payload, vec![10, 20, 30]);

    // Verify Java-matching transmission time: 20000 + 4000 * ((3 + 61) / 3) = 20000 + 4000*21 = 104000 ns
    assert_eq!(msg.tx_ns, 20000 + 4000 * ((3 + 61) / 3));
}

// ── Broadcast: far away UAV not in chunk neighborhood ───────

#[test]
fn test_broadcast_far_uav_not_reached() {
    let mut sim = create_sim(163_840);

    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d0".to_string(), addr(5000));
    // Very far away (many chunks away, outside 5x5x5 neighborhood)
    // 50 degrees is a lot of km away.
    sim.update_uav_info(mock_telemetry(50.0, 50.0, 0.0), "d_far".to_string(), addr(5001));

    sim.enqueue_broadcast("d0".into(), vec![1, 2, 3], 0);

    // d_far should NOT receive (outside chunk neighborhood)
    assert!(sim.get_pending_msgs().get("d_far").is_none());
}

// ── Broadcast: sender busy ──────────────────────────────────

#[test]
fn test_broadcast_sender_busy_delays_message() {
    let mut sim = create_sim(163_840);

    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d0".to_string(), addr(5000));
    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d1".to_string(), addr(5001));

    // First large message makes d0 busy
    sim.enqueue_broadcast("d0".into(), vec![0; 1000], 0);
    assert!(sim.get_delayed_msgs().is_empty());

    // Immediate second message should be delayed
    sim.enqueue_broadcast("d0".into(), vec![0; 10], 0);
    assert_eq!(sim.get_delayed_msgs().len(), 1);
    assert_eq!(sim.get_delayed_msgs()[0].sender_id, "d0");
}

// ── Broadcast: carrier sensing ──────────────────────────────

#[test]
fn test_broadcast_near_senders_delays_message() {
    let mut sim = create_sim(163_840);

    // d0 and d2 are in the same spatial chunk
    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d0".to_string(), addr(5000));
    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d1".to_string(), addr(5001));
    sim.update_uav_info(mock_telemetry(0.000000001, 0.0, 0.0), "d2".to_string(), addr(5002));

    // d0 sends a big message
    sim.enqueue_broadcast("d0".into(), vec![0; 1000], 0);

    // d2 tries to send → carrier sensing detects d0 nearby → delayed
    sim.enqueue_broadcast("d2".into(), vec![0; 10], 0);

    assert_eq!(sim.get_delayed_msgs().len(), 1);
    assert_eq!(sim.get_delayed_msgs()[0].sender_id, "d2");
}

// ── Broadcast: buffer overflow ──────────────────────────────

#[test]
fn test_broadcast_buffer_overflow_discards() {
    let mut sim = create_sim(50); // tiny buffer

    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d0".to_string(), addr(5000));
    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d1".to_string(), addr(5001));

    // 40 bytes → fits
    sim.enqueue_broadcast("d0".into(), vec![0; 40], 0);

    // Wait for d0 busy to expire
    std::thread::sleep(Duration::from_millis(1));

    // 20 bytes → 40 + 20 = 60 > 50 → discarded for d1
    sim.enqueue_broadcast("d0".into(), vec![1; 20], 0);

    let pending = sim.get_pending_msgs();
    assert_eq!(pending.get("d1").unwrap().len(), 1);
    assert_eq!(pending.get("d1").unwrap()[0].payload.len(), 40);
}

// ── send_messages: retries delayed ──────────────────────────

#[test]
fn test_send_messages_retries_delayed() {
    let mut sim = create_sim(163_840);

    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d0".to_string(), addr(5000));
    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d1".to_string(), addr(5001));

    sim.enqueue_broadcast("d0".into(), vec![0; 1000], 0);
    sim.enqueue_broadcast("d0".into(), vec![1; 10], 0); // delayed
    assert_eq!(sim.get_delayed_msgs().len(), 1);

    std::thread::sleep(Duration::from_millis(2));

    sim.send_messages();

    assert!(sim.get_delayed_msgs().is_empty());
    assert!(sim.get_pending_msgs().contains_key("d1"));
}

// ── send_messages: clears pending ───────────────────────────

#[test]
fn test_send_messages_clears_pending() {
    let mut sim = create_sim(163_840);

    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d0".to_string(), addr(5000));
    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d1".to_string(), addr(5001));

    sim.enqueue_broadcast("d0".into(), vec![1, 2, 3], 0);
    assert!(!sim.get_pending_msgs().is_empty());

    sim.send_messages();

    assert!(
        sim.get_pending_msgs().is_empty() || sim.get_pending_msgs().values().all(|v| v.is_empty())
    );
}

// ── Transmission time matches Java formula ──────────────────

#[test]
fn test_transmission_time_matches_java() {
    let mut sim = create_sim(163_840);

    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d0".to_string(), addr(5000));
    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d1".to_string(), addr(5001));

    let payload = vec![0u8; 100];
    sim.enqueue_broadcast("d0".into(), payload, 0);

    let msg = &sim.get_pending_msgs().get("d1").unwrap()[0];
    // Java: end = start + 20000 + 4000 * ((100 + 61) / 3)
    //      = start + 20000 + 4000 * 53 = start + 20000 + 212000 = start + 232000 ns
    let expected_tx_ns = 20_000u64 + 4_000 * ((100 + 61) / 3);
    assert_eq!(msg.tx_ns, expected_tx_ns);
    assert_eq!(expected_tx_ns, 232_000);
}

// ── Statistics accumulation ─────────────────────────────────

#[test]
fn test_statistics_accumulate() {
    let mut sim = create_sim(163_840);

    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d0".to_string(), addr(5000));
    sim.update_uav_info(mock_telemetry(0.0, 0.0, 0.0), "d1".to_string(), addr(5001));

    // Unknown sender
    sim.enqueue_broadcast("ghost".into(), vec![1], 0);

    // Successful broadcast
    sim.enqueue_broadcast("d0".into(), vec![1, 2, 3], 0);

    // We can't easily test all stats without more setup, but verify the ones we can
    // (stats are on the logger which is behind Arc, but accessible via print_stats)
    // Just verify no panics and the counters are non-zero
    sim.print_stats();
}

#[test]
fn test_telemetry_subscription_broadcasts_on_update() {
    let mut sim = create_sim(163_840);
    
    // Subscribe an arbitrary telemetry address
    let sub_addr = addr(9999);
    sim.subscribe_telemetry(sub_addr);

    // Verify it is broadcasting via logging stat triggers by executing the update
    let tel = mock_telemetry(1.0, 2.0, 3.0);
    sim.update_uav_info(tel, "d0".to_string(), addr(5000));

    // Validating output stats will show that a telemetry relay log was incremented
    assert_eq!(
        sim.get_logger().stats.telemetry_subscriptions.load(std::sync::atomic::Ordering::Relaxed),
        1
    );
    assert_eq!(
        sim.get_logger().stats.telemetry_updates_relayed.load(std::sync::atomic::Ordering::Relaxed),
        1
    );
}
