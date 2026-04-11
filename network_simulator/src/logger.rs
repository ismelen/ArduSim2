//! # Logger Module
//!
//! Provides a structured, timestamped console logger and maintains
//! accumulated statistics for the network simulation, tracking broadcasts,
//! discarded messages, and UDP send errors.

use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::time::SystemTime;
use std::fs::{File, OpenOptions};
use std::io::Write;
use std::path::Path;

/// Accumulated statistics for the network simulation pipeline.
#[derive(Debug, Default)]
pub struct Stats {
    /// Total broadcast messages received from UAVs.
    pub total_broadcasts: AtomicU64,
    /// Total messages successfully enqueued for delivery.
    pub total_delivered: AtomicU64,
    /// Messages discarded because the sender was unknown.
    pub discarded_unknown_sender: AtomicU64,
    /// Messages discarded due to simulated distance packet loss.
    pub discarded_distance: AtomicU64,
    /// Messages discarded because the receiving UAV was busy transmitting.
    pub discarded_receiver_busy: AtomicU64,
    /// Messages discarded due to the receiving UAV's buffer being full.
    pub discarded_buffer_full: AtomicU64,
    /// Messages discarded due to temporal collisions at the receiver.
    pub discarded_collision: AtomicU64,
    /// Messages discarded after exceeding the maximum number of CSMA retries.
    pub discarded_max_retries: AtomicU64,
    /// Transmissions delayed because the sender was already busy.
    pub delayed_sender_busy: AtomicU64,
    /// Transmissions delayed due to carrier sensing (nearby UAVs transmitting).
    pub delayed_carrier_sensing: AtomicU64,
    /// Delayed transmissions that were successfully retried.
    pub delayed_retried: AtomicU64,
    /// Successful UDP deliveries to receivers.
    pub udp_send_ok: AtomicU64,
    /// Failed UDP deliveries.
    pub udp_send_err: AtomicU64,
    /// Telemetry subscriptions received.
    pub telemetry_subscriptions: AtomicU64,
    /// Telemetry updates relayed to subscribers.
    pub telemetry_updates_relayed: AtomicU64,
}

impl Stats {
    /// Atomically increments a stat field by 1.
    fn inc(&self, field: &AtomicU64) {
        field.fetch_add(1, Ordering::Relaxed);
    }

    /// Returns a formatted summary of all accumulated statistics.
    pub fn summary(&self) -> String {
        format!(
            "\n=== Network Simulator Statistics ===\n\
             Total broadcasts received:    {}\n\
             Delivered to receivers:        {}\n\
             Discarded (unknown sender):    {}\n\
             Discarded (distance loss):     {}\n\
             Discarded (receiver busy):     {}\n\
             Discarded (buffer full):       {}\n\
             Discarded (collision/overlap): {}\n\
             Discarded (max retries):       {}\n\
             Delayed (sender busy):         {}\n\
             Delayed (carrier sensing):     {}\n\
             Delayed retried:               {}\n\
             UDP send OK:                   {}\n\
             UDP send errors:               {}\n\
             Telemetry subscriptions:       {}\n\
             Telemetry updates relayed:     {}\n\
             ====================================",
            self.total_broadcasts.load(Ordering::Relaxed),
            self.total_delivered.load(Ordering::Relaxed),
            self.discarded_unknown_sender.load(Ordering::Relaxed),
            self.discarded_distance.load(Ordering::Relaxed),
            self.discarded_receiver_busy.load(Ordering::Relaxed),
            self.discarded_buffer_full.load(Ordering::Relaxed),
            self.discarded_collision.load(Ordering::Relaxed),
            self.discarded_max_retries.load(Ordering::Relaxed),
            self.delayed_sender_busy.load(Ordering::Relaxed),
            self.delayed_carrier_sensing.load(Ordering::Relaxed),
            self.delayed_retried.load(Ordering::Relaxed),
            self.udp_send_ok.load(Ordering::Relaxed),
            self.udp_send_err.load(Ordering::Relaxed),
            self.telemetry_subscriptions.load(Ordering::Relaxed),
            self.telemetry_updates_relayed.load(Ordering::Relaxed),
        )
    }
}

/// Representation of message severity for console logging.
pub enum LogLevel {
    Info,
    Success,
    Warning,
    Error,
    Delay,
    Discard,
    SuccessSummary,
}

impl LogLevel {
    /// Returns a string literal representing the severity label.
    fn label(&self) -> &'static str {
        match self {
            LogLevel::Info => "INFO",
            LogLevel::Success => "SUCCESS",
            LogLevel::Warning => "WARNING",
            LogLevel::Error => "ERROR",
            LogLevel::Delay => "DELAY",
            LogLevel::Discard => "DISCARD",
            LogLevel::SuccessSummary => "SUCCESS",
        }
    }

    /// Returns true if this level is considered 'important' logic.
    fn is_important(&self) -> bool {
        matches!(self, LogLevel::Error | LogLevel::Warning | LogLevel::SuccessSummary)
    }
}

pub trait LogWriter: Send + Sync {
    fn write(&self, ts: &str, level: &LogLevel, message: &str);
}

pub struct StdOutWriter;
impl LogWriter for StdOutWriter {
    fn write(&self, ts: &str, level: &LogLevel, message: &str) {
        println!("[{}] [{}] {}", ts, level.label(), message);
    }
}

pub struct UdpWriter {
    socket: std::net::UdpSocket,
    target: String,
}

impl UdpWriter {
    pub fn new(ip: &str, port: u16) -> Self {
        let socket = std::net::UdpSocket::bind("0.0.0.0:0").expect("Failed to bind UDP logger socket");
        Self {
            socket,
            target: format!("{}:{}", ip, port),
        }
    }
}

impl LogWriter for UdpWriter {
    fn write(&self, ts: &str, level: &LogLevel, message: &str) {
        let formatted = format!("[{}] [{}] {}\n", ts, level.label(), message);
        let _ = self.socket.send_to(formatted.as_bytes(), &self.target);
    }
}

pub struct FileWriter {
    file: std::sync::Mutex<File>,
}

impl FileWriter {
    pub fn new<P: AsRef<Path>>(path: P) -> std::io::Result<Self> {
        let file = OpenOptions::new()
            .create(true)
            .append(true)
            .open(path)?;
        Ok(Self {
            file: std::sync::Mutex::new(file),
        })
    }
}

impl LogWriter for FileWriter {
    fn write(&self, ts: &str, level: &LogLevel, message: &str) {
        if let Ok(mut file) = self.file.lock() {
            let _ = writeln!(file, "[{}] [{}] {}", ts, level.label(), message);
        }
    }
}

pub struct MultiWriter {
    writers: Vec<Box<dyn LogWriter>>,
}

impl MultiWriter {
    pub fn new(writers: Vec<Box<dyn LogWriter>>) -> Self {
        Self { writers }
    }
}

impl LogWriter for MultiWriter {
    fn write(&self, ts: &str, level: &LogLevel, message: &str) {
        for writer in &self.writers {
            writer.write(ts, level, message);
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub enum LogMode {
    Debug,
    DebugImportant,
    Prod { ip: String, port: u16 },
}

pub struct LoggerFactory;

impl LoggerFactory {
    pub fn create(mode: LogMode) -> Logger {
        let (writer, mut is_important_only): (Box<dyn LogWriter>, bool) = match mode.clone() {
            LogMode::Debug => (Box::new(StdOutWriter), false),
            LogMode::DebugImportant => (Box::new(StdOutWriter), true),
            LogMode::Prod { ip, port } => (Box::new(UdpWriter::new(&ip, port)), true),
        };

        // Override importance if DEBUG=true env var is set
        if std::env::var("DEBUG").unwrap_or_default() == "true" {
            is_important_only = false;
        }

        // Add file logging if /app/logs directory exists
        let mut final_writer = writer;
        if Path::new("/app/logs").is_dir() {
            if let Ok(file_writer) = FileWriter::new("/app/logs/network_simulator.log") {
                final_writer = Box::new(MultiWriter::new(vec![
                    final_writer,
                    Box::new(file_writer),
                ]));
            }
        }

        let logger = Logger {
            stats: Stats::default(),
            enabled: AtomicBool::new(true),
            writer: final_writer,
            is_important_only,
        };
        
        logger.log(LogLevel::Info, &format!("Network simulator logger initialized in {:?} mode", mode));
        if !is_important_only {
            logger.log(LogLevel::Info, "Verbose logging enabled (DEBUG=true)");
        }
        logger
    }
}

/// Timestamped console logger with integrated statistics tracking.
pub struct Logger {
    /// Global tracking of metrics across the Simulation.
    pub stats: Stats,
    /// Indicates whether console logging is actively printed to standard output.
    enabled: AtomicBool,
    /// Strategy to write logs (e.g. standard output, UDP socket).
    writer: Box<dyn LogWriter>,
    /// Whether to only write important logs.
    is_important_only: bool,
}

impl Logger {
    /// Default creation for tests or backward compatibility.
    pub fn new() -> Self {
        LoggerFactory::create(LogMode::Debug)
    }

    /// Enable or disable log output (statistics are always accumulated regardless).
    pub fn set_enabled(&self, enabled: bool) {
        self.enabled.store(enabled, Ordering::Relaxed);
    }

    /// Logs a message directly with a given severity level.
    pub fn log(&self, level: LogLevel, message: &str) {
        if !self.enabled.load(Ordering::Relaxed) {
            return;
        }
        if self.is_important_only && !level.is_important() {
            return;
        }
        let ts = humanize_timestamp(SystemTime::now());
        self.writer.write(&ts, &level, message);
    }

    /// Logs the registration or position update of a given UAV.
    pub fn uav_registered(&self, uav_id: &str) {
        self.log(
            LogLevel::Info,
            &format!("UAV '{}' registered / position updated", uav_id),
        );
    }

    /// Logs a subscription from a given address.
    pub fn topic_susbscribed(&self, addr: &std::net::SocketAddr, topic: &str) {
        if topic == "telemetry" {
            self.stats.inc(&self.stats.telemetry_subscriptions)
        }
        self.log(
            LogLevel::Success,
            &format!("{} subscriber registered: {}", topic, addr)
        )
    }

    /// Logs when a telemetry update is related to subscribers.
    pub fn telemetry_relayed(&self) {
        self.stats.inc(&self.stats.telemetry_updates_relayed);
        // Do not print every single telemetry relay to avoid spam.
    }

    /// Logs and counts when a broadcast request is received.
    pub fn broadcast_received(&self, sender_id: &str) {
        self.stats.inc(&self.stats.total_broadcasts);
        self.log(LogLevel::Info, &format!("Broadcast from '{}'", sender_id));
    }

    /// Logs and counts successful enqueueing of a broadcast to a receiver.
    pub fn broadcast_enqueued(
        &self,
        _sender_id: &str,
        _receiver_id: &str,
        _payload_len: usize,
        _tx_ns: u64,
    ) {
        self.stats.inc(&self.stats.total_delivered);
        // We no longer log individual successful deliveries here to reduce noise.
    }

    /// Logs a summary of successful deliveries for a single broadcast.
    pub fn broadcast_success_summary(&self, sender_id: &str, num_receivers: usize) {
        if num_receivers > 0 {
            self.log(
                LogLevel::SuccessSummary,
                &format!("Broadcast from '{}' delivered to {} receiver(s)", sender_id, num_receivers),
            );
        }
    }

    /// Logs and counts a delay in transmission due to the sender already being busy.
    pub fn msg_delayed_sender_busy(&self, sender_id: &str) {
        self.stats.inc(&self.stats.delayed_sender_busy);
        self.log(
            LogLevel::Delay,
            &format!("Broadcast from '{}' delayed: sender busy", sender_id),
        );
    }

    /// Logs and counts a transmission delay resulting from simulated carrier sensing.
    pub fn msg_delayed_near_senders(&self, sender_id: &str) {
        self.stats.inc(&self.stats.delayed_carrier_sensing);
        self.log(
            LogLevel::Delay,
            &format!("Broadcast from '{}' delayed: carrier sensing", sender_id),
        );
    }

    /// Logs and counts discarded messages when the sender is unknown.
    pub fn msg_discarded_unknown_sender(&self, sender_id: &str) {
        self.stats.inc(&self.stats.discarded_unknown_sender);
        self.log(
            LogLevel::Discard,
            &format!(
                "Broadcast from '{}' discarded: sender not registered",
                sender_id
            ),
        );
    }

    /// Logs and counts discarded messages originating from simulated distance loss.
    pub fn msg_discarded_distance(&self, sender_id: &str, receiver_id: &str) {
        self.stats.inc(&self.stats.discarded_distance);
        self.log(
            LogLevel::Discard,
            &format!(
                "'{}' -> '{}' discarded: distance loss",
                sender_id, receiver_id
            ),
        );
    }

    /// Logs and counts when the target receiver is already busy receiving another message.
    pub fn msg_discarded_receiver_busy(&self, sender_id: &str, receiver_id: &str) {
        self.stats.inc(&self.stats.discarded_receiver_busy);
        self.log(
            LogLevel::Discard,
            &format!(
                "'{}' -> '{}' discarded: receiver busy",
                sender_id, receiver_id
            ),
        );
    }

    /// Logs and counts message discards attributing to the receiver's simulated buffer maxing out.
    pub fn msg_discarded_buffer_full(&self, sender_id: &str, receiver_id: &str) {
        self.stats.inc(&self.stats.discarded_buffer_full);
        self.log(
            LogLevel::Discard,
            &format!(
                "'{}' -> '{}' discarded: buffer full",
                sender_id, receiver_id
            ),
        );
    }

    /// Logs and counts when overlapping messages inevitably lead to data corruption or a drop.
    pub fn msg_overlapped(&self, receiver_id: &str) {
        self.stats.inc(&self.stats.discarded_collision);
        self.log(
            LogLevel::Warning,
            &format!("Message to '{}' dropped: collision", receiver_id),
        );
    }

    /// Logs and counts messages discarded for exceeding max CSMA retries.
    pub fn msg_discarded_max_retries(&self, sender_id: &str) {
        self.stats.inc(&self.stats.discarded_max_retries);
        self.log(
            LogLevel::Discard,
            &format!("Broadcast from '{}' discarded: max CSMA retries exceeded", sender_id),
        );
    }

    /// Tracks instances where UDP packets were successfully sent.
    pub fn msg_sent_ok(&self, receiver_id: &str) {
        self.stats.inc(&self.stats.udp_send_ok);
        self.log(
            LogLevel::Success,
            &format!("Delivered to '{}'", receiver_id),
        );
    }

    /// Tracks instances where UDP broadcasts failed on actual logical/physical sending.
    pub fn msg_sent_err(&self, receiver_id: &str, err: &str) {
        self.stats.inc(&self.stats.udp_send_err);
        self.log(
            LogLevel::Error,
            &format!("Failed to deliver to '{}': {}", receiver_id, err),
        );
    }

    /// Tracks successfully retried messages that were previously delayed in queue.
    pub fn delayed_queue_retry(&self, count: usize) {
        if count > 0 {
            self.stats
                .total_broadcasts
                .fetch_add(count as u64, Ordering::Relaxed);
            self.stats
                .delayed_retried
                .fetch_add(count as u64, Ordering::Relaxed);
            self.log(
                LogLevel::Info,
                &format!("Retrying {} delayed message(s)", count),
            );
        }
    }

    /// Convenient call to systematically output the stored accumulated summary statistics.
    pub fn print_stats(&self) {
        println!("{}", self.stats.summary());
    }
}

/// Converts timestamp objects securely into human readable formatting.
fn humanize_timestamp(time: SystemTime) -> String {
    let duration = time
        .duration_since(SystemTime::UNIX_EPOCH)
        .unwrap_or_default();
    let secs = duration.as_secs();
    let millis = duration.subsec_millis();
    let s = secs % 60;
    let m = (secs / 60) % 60;
    let h = (secs / 3600) % 24;
    format!("{:02}:{:02}:{:02}.{:03}", h, m, s, millis)
}
