use std::time::SystemTime;

pub enum LogLevel {
  Info,
  Success,
  Warning,
  Error,
  Delay,
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

pub struct Logger;

impl Logger {
  pub fn new() -> Self {
    let logger = Self;
    logger.log(LogLevel::Info, "Logger initialized");
    logger
  }

  pub fn log(&self, level: LogLevel, message: &str) {
    let timestamp = humanize_timestamp(SystemTime::now());
    println!("[{}] [{}] {}", timestamp, level.label(), message);
  }

  // --- convenience methods ---

  pub fn info(&self, message: &str) {
    self.log(LogLevel::Info, message);
  }

  pub fn success(&self, message: &str) {
    self.log(LogLevel::Success, message);
  }

  pub fn warning(&self, message: &str) {
    self.log(LogLevel::Warning, message);
  }

  pub fn error(&self, message: &str) {
    self.log(LogLevel::Error, message);
  }

  pub fn delay(&self, message: &str) {
    self.log(LogLevel::Delay, message);
  }

  pub fn discard(&self, message: &str) {
    self.log(LogLevel::Discard, message);
  }

  // --- domain-specific helpers ---

  pub fn uav_registered(&self, uav_id: &str) {
    self.info(&format!("UAV '{}' registered / position updated", uav_id));
  }

  pub fn msg_delayed_sender_busy(&self, sender_id: &str, target_id: &str) {
    self.delay(&format!(
      "Message from '{}' to '{}' delayed: sender is busy transmitting",
      sender_id, target_id
    ));
  }

  pub fn msg_delayed_near_senders(&self, sender_id: &str, target_id: &str) {
    self.delay(&format!(
      "Message from '{}' to '{}' delayed: nearby UAVs are transmitting (collision avoidance)",
      sender_id, target_id
    ));
  }

  pub fn msg_discarded_distance(&self, sender_id: &str, target_id: &str) {
    self.discard(&format!(
      "Message from '{}' to '{}' discarded: distance loss probability exceeded",
      sender_id, target_id
    ));
  }

  pub fn msg_discarded_unknown_sender(&self, sender_id: &str, target_id: &str) {
    self.discard(&format!(
      "Message from '{}' to '{}' discarded: sender not registered",
      sender_id, target_id
    ));
  }

  pub fn msg_discarded_unknown_target(&self, sender_id: &str, target_id: &str) {
    self.discard(&format!(
      "Message from '{}' to '{}' discarded: target not registered",
      sender_id, target_id
    ));
  }

  pub fn msg_enqueued(&self, sender_id: &str, target_id: &str, payload_len: usize, tx_us: u64) {
    self.info(&format!(
      "Message from '{}' to '{}' enqueued ({} bytes, tx={}µs)",
      sender_id, target_id, payload_len, tx_us
    ));
  }

  pub fn msg_overlapped(&self, target_id: &str) {
    self.warning(&format!(
      "Message to '{}' dropped due to signal overlap (collision)",
      target_id
    ));
  }

  pub fn msg_sent_ok(&self, target_id: &str) {
    self.success(&format!("Message delivered to '{}'", target_id));
  }

  pub fn msg_sent_err(&self, target_id: &str, err: &str) {
    self.error(&format!("Failed to deliver message to '{}': {}", target_id, err));
  }

  pub fn delayed_queue_retry(&self, count: usize) {
    if count > 0 {
      self.info(&format!("Retrying {} delayed message(s)", count));
    }
  }
}

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
