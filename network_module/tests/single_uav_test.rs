use std::{ net::SocketAddr, time::{Duration, SystemTime, UNIX_EPOCH}};

use network_module::network_manager::{models::{Coords, Message}, network_manager::NetworkManager, uav_manager::UAVManager};
use tokio::time::sleep;


#[tokio::test]
async fn first_message_should_save_uav_data() {
  const UAV_ID: &str = "1";  
  const UAV_COORDS: Option<Coords> = Some(Coords{
    x: 1.0,
    y: 2.0,
    z: 3.0,
  });
  let uav_addr: SocketAddr = "127.0.0.1:5000".parse().unwrap();

  let uav_manager = UAVManager::new();
  let network_manager = NetworkManager::new(&uav_manager);

  let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_micros();
  
  let message = Message{
    sender_id: UAV_ID.to_string(),
    start: now,
    end: None,
    content: None,
    coords: UAV_COORDS,
    receiver_id: None
  };

  let json_msg = serde_json::to_value(&message).unwrap();
  
  network_manager.push(json_msg, uav_addr).await;
  sleep(Duration::from_millis(100)).await;

  let stored_addr = uav_manager.get_addr(UAV_ID);
  let stored_coords = uav_manager.get_coords(UAV_ID);
  assert_eq!(stored_addr, Some(uav_addr));
  assert_eq!(stored_coords, UAV_COORDS);
}
