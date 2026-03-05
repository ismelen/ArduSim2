use std::time::SystemTime;

/// Severity levels for log messages.
///
/// Each level is displayed with a distinct label in the log output,
/// making it easy to filter and identify message types.
pub enum LogLevel {
  /// General informational messages (e.g., UAV registration, message enqueued).
  Info,
  /// Successful operations (e.g., message delivered).
  Success,
  /// Non-critical issues (e.g., signal overlap/collision).
  Warning,
  /// Errors (e.g., failed UDP delivery).
  Error,
  /// Messages that were delayed due to busy sender or nearby transmissions.
  Delay,
  /// Messages that were discarded (e.g., unknown sender/target, distance loss).
  Discard,
}

impl LogLevel {
  fn label(&self) -> &'static str {
    match self {
      LogLevel::Info    => "INFO",
      LogLevel::Success => "SUCCESS",
      LogLevel::Warning => "WARNING",
      LogLevel::Error   => "ERROR",
      LogLevel::Delay   => "DELAY",
      LogLevel::Discard => "DISCARD",
    }
  }
}

/// Console logger for the network simulator.
///
/// Outputs timestamped, level-tagged messages to stdout in the format:
/// ```text
/// [HH:MM:SS.mmm] [LEVEL] message
/// ```
///
/// Provides domain-specific helper methods for common simulator events
/// such as UAV registration, message enqueuing, delivery, and discarding.
pub struct Logger;

impl Logger {
  /// Creates a new logger and prints an initialization message.
  pub fn new() -> Self {
    let logger = Self;
    logger.log(LogLevel::Info, "Logger initialized");
    logger
  }

  /// Logs a message with the given level and a human-readable timestamp.
  pub fn log(&self, level: LogLevel, message: &str) {
    let timestamp = humanize_timestamp(SystemTime::now());
    println!("[{}] [{}] {}", timestamp, level.label(), message);
  }

  // --- convenience methods ---

  /// Logs an informational message.
  pub fn info(&self, message: &str) {
    self.log(LogLevel::Info, message);
  }

  /// Logs a success message.
  pub fn success(&self, message: &str) {
    self.log(LogLevel::Success, message);
  }

  /// Logs a warning message.
  pub fn warning(&self, message: &str) {
    self.log(LogLevel::Warning, message);
  }

  /// Logs an error message.
  pub fn error(&self, message: &str) {
    self.log(LogLevel::Error, message);
  }

  /// Logs a delay message.
  pub fn delay(&self, message: &str) {
    self.log(LogLevel::Delay, message);
  }

  /// Logs a discard message.
  pub fn discard(&self, message: &str) {
    self.log(LogLevel::Discard, message);
  }

  // --- domain-specific helpers ---

  /// Logs that a UAV has been registered or its position updated.
  pub fn uav_registered(&self, uav_id: &str) {
    self.info(&format!("UAV '{}' registered / position updated", uav_id));
  }

  /// Logs that a message was delayed because the sender is still busy transmitting.
  pub fn msg_delayed_sender_busy(&self, sender_id: &str, target_id: &str) {
    self.delay(&format!(
      "Message from '{}' to '{}' delayed: sender is busy transmitting",
      sender_id, target_id
    ));
  }

  /// Logs that a message was delayed due to nearby UAVs actively transmitting (collision avoidance).
  pub fn msg_delayed_near_senders(&self, sender_id: &str, target_id: &str) {
    self.delay(&format!(
      "Message from '{}' to '{}' delayed: nearby UAVs are transmitting (collision avoidance)",
      sender_id, target_id
    ));
  }

  /// Logs that a message was discarded because the distance loss probability was exceeded.
  pub fn msg_discarded_distance(&self, sender_id: &str, target_id: &str) {
    self.discard(&format!(
      "Message from '{}' to '{}' discarded: distance loss probability exceeded",
      sender_id, target_id
    ));
  }

  /// Logs that a message was discarded because the sender is not registered.
  pub fn msg_discarded_unknown_sender(&self, sender_id: &str, target_id: &str) {
    self.discard(&format!(
      "Message from '{}' to '{}' discarded: sender not registered",
      sender_id, target_id
    ));
  }

  /// Logs that a message was discarded because the target is not registered.
  pub fn msg_discarded_unknown_target(&self, sender_id: &str, target_id: &str) {
    self.discard(&format!(
      "Message from '{}' to '{}' discarded: target not registered",
      sender_id, target_id
    ));
  }

  /// Logs that a message has been enqueued for delivery.
  pub fn msg_enqueued(&self, sender_id: &str, target_id: &str, payload_len: usize, tx_us: u64) {
    self.info(&format!(
      "Message from '{}' to '{}' enqueued ({} bytes, tx={}µs)",
      sender_id, target_id, payload_len, tx_us
    ));
  }

  /// Logs that a message was dropped due to signal overlap (collision) at the receiver.
  pub fn msg_overlapped(&self, target_id: &str) {
    self.warning(&format!(
      "Message to '{}' dropped due to signal overlap (collision)",
      target_id
    ));
  }

  /// Logs successful message delivery.
  pub fn msg_sent_ok(&self, target_id: &str) {
    self.success(&format!("Message delivered to '{}'", target_id));
  }

  /// Logs a failed message delivery attempt.
  pub fn msg_sent_err(&self, target_id: &str, err: &str) {
    self.error(&format!("Failed to deliver message to '{}': {}", target_id, err));
  }

  /// Logs the retry of delayed messages from the queue.
  pub fn delayed_queue_retry(&self, count: usize) {
    if count > 0 {
      self.info(&format!("Retrying {} delayed message(s)", count));
    }
  }
}

/// Converts a [`SystemTime`] into a human-readable `HH:MM:SS.mmm` string (UTC).
fn humanize_timestamp(time: SystemTime) -> String {
  let duration = time.duration_since(SystemTime::UNIX_EPOCH).unwrap_or_default();
  let secs = duration.as_secs();
  let millis = duration.subsec_millis();

  let total_mins = secs / 60;
  let s = secs % 60;
  let total_hours = total_mins / 60;
  let m = total_mins % 60;
  let h = total_hours % 24;

  format!("{:02}:{:02}:{:02}.{:03}", h, m, s, millis)
}
