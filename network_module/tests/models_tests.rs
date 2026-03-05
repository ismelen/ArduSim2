use network_module::models::{Message, Position};
use std::time::Instant;

#[test]
fn test_position_eq() {
    let p1 = Position { x: 1.0, y: 2.0, z: 3.0 };
    let p2 = Position { x: 1.0, y: 2.0, z: 3.0 };
    let p3 = Position { x: 1.1, y: 2.0, z: 3.0 };
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
    assert_eq!(m.sender_id, "".to_string());
    assert_eq!(m.target_id, "".to_string());
    assert_eq!(m.tx, 0);
    assert_eq!(m.overlapped, false);
    assert!(m.target_addr.is_none());
}

#[test]
fn test_message_ordering() {
    let mut m1 = Message::default();
    let mut m2 = Message::default();
    
    m1.from = Instant::now();
    m2.from = m1.from + std::time::Duration::from_millis(10);
    
    assert!(m1 < m2);
    assert!(m2 > m1);
    
    m2.from = m1.from;
    m2.to = m1.to;
    assert_eq!(m1, m2);
}
