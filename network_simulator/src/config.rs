use std::time::Duration;

/// Default receiving buffer size in bytes.
pub const DEFAULT_BUFFER_SIZE: usize = 163_840;

/// Size of each spatial chunk in meters.
pub const CHUNK_SIZE_M: f64 = 350.0;


/// Neighborhood radius for chunk lookups (5x5x5 = offsets -2..+2).
pub const CHUNK_RADIUS: i64 = 2;

/// Maximum number of CSMA retries before discarding a delayed broadcast.
pub const MAX_CSMA_RETRIES: u32 = 5;

/// Maximum physical range of signal mapping (in meters) - mapped dynamically over squared logic to avoid roots.
pub const MAX_RANGE_M: f64 = 1350.0;

/// Max UDP datagram size payload bounds safely expected through networks.
pub const MAX_DATAGRAM_SIZE: usize = 65507;

/// Interval between message flush cycles natively processed by the async dispatcher loop.
pub const SWITCH_INTERVAL: Duration = Duration::from_millis(1);

/// Ratio to translate raw longitude/latitude degrees into meter offsets.
pub const METERS_PER_DEGREE: f64 = 111.12 * 1000.0;