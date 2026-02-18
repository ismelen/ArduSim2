use std::{collections::HashMap, sync::{Arc, RwLock}};

use crate::network_manager::models::{BusyState, Coords};

#[derive(Debug, Clone)]
pub struct UAVManager {
  coords: Arc<RwLock<HashMap<String, Coords>>>,
  busy_states: Arc<RwLock<HashMap<String, BusyState>>>,
  addrs: Arc<RwLock<HashMap<String, String>>>,
}

impl UAVManager {
  pub fn new() -> Self {
    UAVManager {
      coords: Arc::new(RwLock::new(HashMap::new())),
      busy_states: Arc::new(RwLock::new(HashMap::new())),
      addrs: Arc::new(RwLock::new(HashMap::new())),
    }
  }

  pub fn add_uav(&self, uav_id: &str, coords: Coords, addr: String) {
    self.coords.write().unwrap().insert(uav_id.to_string(), coords);
    self.busy_states.write().unwrap().insert(uav_id.to_string(), BusyState { from: 0, to: 0, overlapped: false });
    self.addrs.write().unwrap().insert(uav_id.to_string(), addr);
  }

  pub fn get_addr(&self, uav_id: &str) -> Option<String> {
    self.addrs.read().unwrap().get(uav_id).cloned()
  }

  pub fn update_coords(&self, uav_id: &str, coords: Coords) {
    self.coords.write().unwrap().insert(uav_id.to_string(), coords);
  }

  pub fn get_coords(&self, uav_id: &str) -> Option<Coords> {
    self.coords.read().unwrap().get(uav_id).cloned()
  }

  pub fn update_busy_state(&self, uav_id: &str, new_state: BusyState) {
    self.busy_states.write().unwrap().insert(uav_id.to_string(), new_state);
  }


  pub fn get_busy_state(&self, uav_id: &str) -> Option<BusyState> {
    self.busy_states.read().unwrap().get(uav_id).cloned()
  }
}




