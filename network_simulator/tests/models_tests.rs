use network_simulator::models::{Message, Position};
use std::sync::Arc;

#[test]
fn test_position_equality() {
    let p1 = Position {
        x: 1.0,
        y: 2.0,
        z: 3.0,
    };
    let p2 = Position {
        x: 1.0,
        y: 2.0,
        z: 3.0,
    };
    let p3 = Position {
        x: 1.1,
        y: 2.0,
        z: 3.0,
    };
    assert_eq!(p1, p2);
    assert_ne!(p1, p3);
}

#[test]
fn test_position_default() {
    let p = Position::default();
    assert_eq!(p.x, 0.0);
    assert_eq!(p.y, 0.0);
    assert_eq!(p.z, 0.0);
}

#[test]
fn test_message_default() {
    let m = Message::default();
    assert_eq!(m.sender_id, "");
    assert!(m.payload.is_empty());
    assert_eq!(m.tx_ns, 0);
    assert!(!m.overlapped);
    assert!(m.target_addr.is_none());
}

#[test]
fn test_message_ordering() {
    let mut m1 = Message::default();
    let mut m2 = Message::default();

    m1.from = std::time::Instant::now();
    m2.from = m1.from + std::time::Duration::from_millis(10);

    assert!(m1 < m2);
    assert!(m2 > m1);
}

#[test]
fn test_message_arc_payload_shared() {
    let data = Arc::new(vec![1u8, 2, 3, 4, 5]);
    let m1 = Message {
        payload: Arc::clone(&data),
        ..Default::default()
    };
    let m2 = Message {
        payload: Arc::clone(&data),
        ..Default::default()
    };

    // Both messages share the same underlying allocation
    assert!(Arc::ptr_eq(&m1.payload, &m2.payload));
    assert_eq!(Arc::strong_count(&data), 3);
}
