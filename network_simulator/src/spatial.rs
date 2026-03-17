use std::collections::HashMap;
use std::time::Instant;

use rand::Rng;

use crate::config::{CHUNK_RADIUS, CHUNK_SIZE_M, MAX_RANGE_M};
use crate::models::Position;

/// Manages spatial indexing of UAVs using 3D chunking mechanics over a 0.350km size resolution.
pub struct SpatialGrid {
    /// Spatial index mapping a chunk (x,y,z) to UAVs active in that region.
    chunks: HashMap<(i64, i64, i64), Vec<String>>,
    /// Details of active broadcast lengths across chunks natively for CSMA collision mappings.
    active_transmissions: HashMap<(i64, i64, i64), Instant>,
}

impl SpatialGrid {
    pub fn new() -> Self {
        Self {
            chunks: HashMap::new(),
            active_transmissions: HashMap::new(),
        }
    }

    /// Translates Cartesian spatial positions into grid indices mapping grouping chunks.
    pub fn get_chunk_key(pos: &Position) -> (i64, i64, i64) {
        (
            (pos.x / CHUNK_SIZE_M).floor() as i64,
            (pos.y / CHUNK_SIZE_M).floor() as i64,
            (pos.z / CHUNK_SIZE_M).floor() as i64,
        )
    }

    /// Evaluates existing chunks and registers UAV placement updates when position offsets shift.
    pub fn update_uav_chunk(&mut self, uav_id: &str, old_chunk: Option<(i64, i64, i64)>, new_chunk: (i64, i64, i64)) {
        if let Some(old) = old_chunk {
            if old != new_chunk {
                if let Some(ids) = self.chunks.get_mut(&old) {
                    ids.retain(|id| id != uav_id);
                    if ids.is_empty() {
                        self.chunks.remove(&old);
                    }
                }
                self.chunks.entry(new_chunk).or_default().push(uav_id.to_string());
            }
        } else {
            self.chunks.entry(new_chunk).or_default().push(uav_id.to_string());
        }
    }

    /// Finds associated system nodes located sharing matching scopes ignoring particular ID tags.
    pub fn get_nearby_uav_ids(&self, center: &(i64, i64, i64), exclude_id: &str) -> Vec<String> {
        let mut result = Vec::new();
        for dx in -CHUNK_RADIUS..=CHUNK_RADIUS {
            for dy in -CHUNK_RADIUS..=CHUNK_RADIUS {
                for dz in -CHUNK_RADIUS..=CHUNK_RADIUS {
                    let key = (center.0 + dx, center.1 + dy, center.2 + dz);
                    if let Some(ids) = self.chunks.get(&key) {
                        for id in ids {
                            if id != exclude_id {
                                result.push(id.clone());
                            }
                        }
                    }
                }
            }
        }
        result
    }

    /// Appends busy timeline offsets natively across active transmission tables mapping nearby chunks.
    pub fn record_transmission(&mut self, chunk_key: &(i64, i64, i64), busy_until: Instant) {
        self.active_transmissions
            .entry(*chunk_key)
            .and_modify(|current| {
                if busy_until > *current {
                    *current = busy_until;
                }
            })
            .or_insert(busy_until);
    }

    /// Runs logical pass to ensure localized nodes sit silent safely across their boundaries currently.
    pub fn has_near_senders(&mut self, center: &(i64, i64, i64), now: &Instant) -> bool {
        for dx in -CHUNK_RADIUS..=CHUNK_RADIUS {
            for dy in -CHUNK_RADIUS..=CHUNK_RADIUS {
                for dz in -CHUNK_RADIUS..=CHUNK_RADIUS {
                    let key = (center.0 + dx, center.1 + dy, center.2 + dz);
                    
                    let mut is_expired = false;
                    if let Some(&busy_until) = self.active_transmissions.get(&key) {
                        if busy_until > *now {
                            return true;
                        }
                        is_expired = true;
                    }
                    
                    if is_expired {
                        self.active_transmissions.remove(&key);
                    }
                }
            }
        }
        false
    }

    /// Employs probabilistic failure checks over node distance logic limits based on 5GHz signal drops.
    pub fn pass_distance_check(sender_pos: &Position, receiver_pos: &Position) -> bool {
        let dx = receiver_pos.x - sender_pos.x;
        let dy = receiver_pos.y - sender_pos.y;
        let dz = receiver_pos.z - sender_pos.z;
        let d_sq = dx * dx + dy * dy + dz * dz;

        let max_range_sq = MAX_RANGE_M * MAX_RANGE_M;
        if d_sq > max_range_sq {
            return false;
        }

        let d = d_sq.sqrt();
        let loss_prob = 5.335e-7 * d * d + 3.395e-5 * d;
        let rand_val: f64 = rand::thread_rng().gen();

        rand_val > loss_prob
    }
    
    pub fn get_chunks(&self) -> &HashMap<(i64, i64, i64), Vec<String>> {
        &self.chunks
    }
    
    pub fn get_active_transmissions(&self) -> &HashMap<(i64, i64, i64), Instant> {
        &self.active_transmissions
    }
}
