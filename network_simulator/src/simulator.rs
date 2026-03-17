use std::{
    collections::{HashMap, VecDeque},
    mem,
    net::{SocketAddr, UdpSocket},
    sync::Arc,
    time::{Duration, Instant},
};

use crate::config::{MAX_CSMA_RETRIES};
use crate::dispatcher::UdpDispatcher;
use crate::logger::Logger;
use crate::models::{Message, TelemetryData, UAV};
use crate::spatial::SpatialGrid;
use crate::telemetry::TelemetryManager;

/// Core network simulator. Receives broadcast messages from UAVs via UDP,
/// applies wireless simulation constraints, and forwards them to all other UAVs.
///
/// Refactored to coordinate modularly across internal logic systems.
pub struct NetworkSimulator {
    /// Registered UAVs mapped by their ID.
    pub uavs: HashMap<String, UAV>,
    /// Queue covering messages currently stalled by busy network paths.
    pub delayed_msgs: VecDeque<Message>,
    /// Maps receivers to payloads successfully cleared to be handled over the persistent thread limits.
    pub pending_msgs: HashMap<String, Vec<Message>>,
    /// Limit for how much data nodes can hold prior to dropping new packets.
    pub receiving_buffer_size: usize,

    /// Grid spatial logics managing collision paths logically.
    pub spatial: SpatialGrid,
    /// System routing data globally across nodes.
    pub telemetry: TelemetryManager,
    /// Execution handlers off the core application threads.
    pub dispatcher: UdpDispatcher,
    /// Output system shared securely across system routines for tracking occurrences.
    pub logger: Arc<Logger>,
}

impl NetworkSimulator {
    /// Creates a new network simulator instance initialized using specific buffers.
    pub fn new(send_port: &str, receiving_buffer_size: usize, logger: Arc<Logger>) -> Self {
        let socket = Arc::new(
            UdpSocket::bind(format!("0.0.0.0:{}", send_port)).expect("couldn't bind send socket"),
        );

        let dispatcher = UdpDispatcher::new(socket.clone(), logger.clone());
        let telemetry = TelemetryManager::new(socket, logger.clone());
        let spatial = SpatialGrid::new();

        Self {
            uavs: HashMap::new(),
            delayed_msgs: VecDeque::new(),
            pending_msgs: HashMap::new(),
            receiving_buffer_size,
            spatial,
            telemetry,
            dispatcher,
            logger,
        }
    }

    /// Registers a new subscriber for telemetry updates.
    pub fn subscribe_telemetry(&mut self, addr: SocketAddr) {
        self.telemetry.subscribe(addr);
    }

    /// Informs the simulator of a system active via specific addresses. Registers its updated chunk.
    pub fn update_uav_info(&mut self, telemetry_data: TelemetryData, addr: SocketAddr) {
        let uav_id = telemetry_data.sender_id.clone();
        let position = telemetry_data.position.to_sim_position();
        let new_chunk = SpatialGrid::get_chunk_key(&position);
        
        // Cache the old chunk for updating the spatial grid dynamically.
        let old_chunk = if let Some(existing) = self.uavs.get_mut(&uav_id) {
            let chunk = existing.chunk_key;
            existing.position = position.clone();
            existing.addr = addr;
            existing.chunk_key = new_chunk;
            Some(chunk)
        } else {
            self.uavs.insert(
                uav_id.clone(),
                UAV {
                    id: uav_id.clone(),
                    position: position.clone(),
                    busy_until: Instant::now(),
                    addr,
                    buffer_used: 0,
                    chunk_key: new_chunk,
                },
            );
            None
        };

        self.spatial.update_uav_chunk(&uav_id, old_chunk, new_chunk);
        self.telemetry.relay_telemetry(&telemetry_data);

        self.logger.uav_registered(&uav_id);
    }

    /// Enqueues a message logically routing it through the simulated airwaves.
    pub fn enqueue_broadcast(&mut self, sender_id: String, payload: Vec<u8>, retries: u32) {
        let (sender_pos, sender_busy_until, sender_chunk) = match self.uavs.get(&sender_id) {
            Some(sender) => (sender.position.clone(), sender.busy_until, sender.chunk_key),
            None => {
                println!("DEBUG: UNKNOWN SENDER: {}", sender_id);
                self.logger.msg_discarded_unknown_sender(&sender_id);
                return;
            }
        };

        self.logger.broadcast_received(&sender_id);
        let now = Instant::now();

        // 1. Check if sender itself is busy transmitting
        if self.handle_sender_busy(&sender_id, &payload, retries, now, sender_busy_until) { return; }

        // 2. CSMA / Carrier sensing
        if self.handle_carrier_sensing(&sender_id, &payload, retries, now, &sender_chunk) { return; }

        let tx_ns = 20_000u64 + 4_000u64 * ((payload.len() as u64 + 61) / 3);
        let busy_until = now + Duration::from_nanos(tx_ns);

        self.spatial.record_transmission(&sender_chunk, busy_until);

        if let Some(sender) = self.uavs.get_mut(&sender_id) {
            sender.busy_until = busy_until;
        }

        let payload_arc = Arc::new(payload);
        let payload_len = payload_arc.len();

        let receiver_ids = self.spatial.get_nearby_uav_ids(&sender_chunk, &sender_id);
        println!("DEBUG: Enqueue from {} chunk {:?}. Found {} receivers", sender_id, sender_chunk, receiver_ids.len());

        let mut delivered_count = 0;
        for receiver_id in receiver_ids {
            let delivered = self.process_receiver(
                &sender_id,
                &receiver_id,
                &sender_pos,
                &payload_arc,
                payload_len,
                now,
                busy_until,
                tx_ns,
            );
            if delivered {
                delivered_count += 1;
            }
        }
        
        self.logger.broadcast_success_summary(&sender_id, delivered_count);
    }

    /// Extracted check for sender busy rules. Returns true if processing was interrupted (delayed/dropped).
    fn handle_sender_busy(&mut self, sender_id: &String, payload: &[u8], retries: u32, now: Instant, busy_until: Instant) -> bool {
        if now < busy_until {
            if retries >= MAX_CSMA_RETRIES {
                self.logger.msg_discarded_max_retries(sender_id);
            } else {
                self.logger.msg_delayed_sender_busy(sender_id);
                self.delayed_msgs.push_back(Message {
                    sender_id: sender_id.clone(),
                    payload: Arc::new(payload.to_vec()),
                    from: now,
                    retries,
                    ..Default::default()
                });
            }
            return true;
        }
        false
    }
    
    /// Extracted carrier sense. Returns true if CSMA aborted transmit to queue/drop.
    fn handle_carrier_sensing(&mut self, sender_id: &String, payload: &[u8], retries: u32, now: Instant, sender_chunk: &(i64, i64, i64)) -> bool {
        if self.spatial.has_near_senders(sender_chunk, &now) {
            if retries >= MAX_CSMA_RETRIES {
                self.logger.msg_discarded_max_retries(sender_id);
            } else {
                self.logger.msg_delayed_near_senders(sender_id);
                self.delayed_msgs.push_back(Message {
                    sender_id: sender_id.clone(),
                    payload: Arc::new(payload.to_vec()),
                    from: now,
                    retries,
                    ..Default::default()
                });
            }
            return true;
        }
        false
    }

    /// Wraps logical checks required to clear receivers per node in broadcast paths.
    fn process_receiver(
        &mut self,
        sender_id: &str,
        receiver_id: &str,
        sender_pos: &crate::models::Position,
        payload_arc: &Arc<Vec<u8>>,
        payload_len: usize,
        now: Instant,
        busy_until: Instant,
        tx_ns: u64,
    ) -> bool {
        // Safe unwrap because the id comes directly out of the matched node pool.
        let receiver = self.uavs.get(receiver_id).unwrap();

        if !SpatialGrid::pass_distance_check(sender_pos, &receiver.position) {
            self.logger.msg_discarded_distance(sender_id, receiver_id);
            return false;
        }

        if now < receiver.busy_until {
            self.logger.msg_discarded_receiver_busy(sender_id, receiver_id);
            return false;
        }

        if receiver.buffer_used + payload_len > self.receiving_buffer_size {
            self.logger.msg_discarded_buffer_full(sender_id, receiver_id);
            return false;
        }

        let msg = Message {
            sender_id: sender_id.to_string(),
            payload: Arc::clone(payload_arc),
            from: now,
            to: busy_until,
            tx_ns,
            overlapped: false,
            target_addr: Some(receiver.addr),
            retries: 0,
        };

        self.pending_msgs
            .entry(receiver_id.to_string())
            .or_default()
            .push(msg);

        if let Some(r) = self.uavs.get_mut(receiver_id) {
            r.buffer_used += payload_len;
        }

        self.logger.broadcast_enqueued(sender_id, receiver_id, payload_len, tx_ns);
        true
    }

    /// Evaluates stored messages logically handing off clean packets mapping their collision scopes.
    pub fn send_messages(&mut self) {
        if !self.pending_msgs.is_empty() {
            let pending = mem::take(&mut self.pending_msgs);

            for (receiver_id, msgs) in &pending {
                let total: usize = msgs.iter().map(|m| m.payload.len()).sum();
                if let Some(uav) = self.uavs.get_mut(receiver_id) {
                    uav.buffer_used = uav.buffer_used.saturating_sub(total);
                }
            }

            self.dispatcher.send_messages(pending);
        }

        let delayed_count = self.delayed_msgs.len();
        self.logger.delayed_queue_retry(delayed_count);

        let old_delayed: VecDeque<Message> = mem::take(&mut self.delayed_msgs);
        for msg in old_delayed {
            self.enqueue_broadcast(msg.sender_id, (*msg.payload).clone(), msg.retries + 1);
        }
    }

    /// Returns a direct system reference for isolated queries to specific elements on record matching IDs.
    pub fn get_uav(&self, uav_id: &str) -> Option<&UAV> {
        self.uavs.get(uav_id)
    }

    /// Grabs read access pointing over active nodes.
    pub fn get_uavs(&self) -> &HashMap<String, UAV> {
        &self.uavs
    }

    /// Allows viewing internally stashed queue elements.
    pub fn get_delayed_msgs(&self) -> &VecDeque<Message> {
        &self.delayed_msgs
    }

    /// Traces waiting deliverables yet to handle evaluation mappings.
    pub fn get_pending_msgs(&self) -> &HashMap<String, Vec<Message>> {
        &self.pending_msgs
    }

    /// Provides access to the internal logger and stats.
    pub fn get_logger(&self) -> &Logger {
        &self.logger
    }

    /// Traces system tracking blocking logic checks on current environment timelines.
    pub fn get_active_transmissions(&self) -> &HashMap<(i64, i64, i64), Instant> {
        self.spatial.get_active_transmissions()
    }

    /// System tracking mapping arrays over grid indices.
    pub fn get_chunks(&self) -> &HashMap<(i64, i64, i64), Vec<String>> {
        self.spatial.get_chunks()
    }

    /// Exposes manual calls down to the integrated analytics module natively printing values inline.
    pub fn print_stats(&self) {
        self.logger.print_stats();
    }
}
