import React, { useEffect, useState } from 'react';
import './SimulationView.css';
import { useAppStore } from './store';

const SimulationView: React.FC = () => {
  const exitSim = useAppStore(s => s.exitSimulation);
  const [time, setTime] = useState(0);

  // Simple timer for SIM_TIME
  useEffect(() => {
    const interval = setInterval(() => setTime(t => t + 1), 1000);
    return () => clearInterval(interval);
  }, []);

  const formatTime = (seconds: number) => {
    const h = Math.floor(seconds / 3600).toString().padStart(2, '0');
    const m = Math.floor((seconds % 3600) / 60).toString().padStart(2, '0');
    const s = (seconds % 60).toString().padStart(2, '0');
    return `${h}:${m}:${s}`;
  };

  return (
    <div className="sim-container">
      {/* ── TOP CONTROLS ── */}
      <div className="sim-top-bar">
        <div className="control-group">
          <button className="control-btn">
            <span className="material-symbols-outlined">pause</span> PAUSE
          </button>
          <button className="control-btn">
            <span className="material-symbols-outlined">stop</span> STOP
          </button>
          <button className="control-btn emergency">
            <span className="material-symbols-outlined">priority_high</span> EMERGENCY RTL
          </button>
        </div>
        <div className="view-toggle">
          <button className="toggle-btn active">2D</button>
          <button className="toggle-btn">3D</button>
        </div>
      </div>

      {/* ── MAP AREA (CENTER) ── */}
      <div className="sim-map-area">
        <div className="map-placeholder">
          {/* In a real app, this would be a Mapbox/Leaflet instance */}
          <div className="map-grid-overlay"></div>
          <div className="uav-ping" style={{ top: '40%', left: '45%' }}></div>
          <div className="uav-ping" style={{ top: '35%', left: '55%' }}></div>
        </div>
      </div>

      {/* ── MISSION LOG (BOTTOM LEFT) ── */}
      <div className="sim-log-panel">
        <div className="panel-header">
          <span className="material-symbols-outlined">terminal</span> MISSION_LOG_ALPHA
          <div className="panel-status">
            <span className="status-indicator stable">UPLINK_STABLE</span>
            <span className="status-indicator sync">SWARM_SYNC</span>
          </div>
        </div>
        <div className="log-content">
          <div className="log-entry">
            <span className="log-time">12:04:12</span>
            <span className="log-level ok">[OK]</span> DRONE_SWARM_01: TARGET_VECTOR_LOCKED // COORDINATES: 36.1699° N, 115.1398° W
          </div>
          <div className="log-entry">
            <span className="log-time">12:04:08</span>
            <span className="log-level info">[INFO]</span> ATMOSPHERIC_COMPENSATION: ACTIVE // WIND_SPEED: 4.2 KTS SE
          </div>
          <div className="log-entry">
            <span className="log-time">12:04:01</span>
            <span className="log-level ok">[OK]</span> FLEET_COMM_ESTABLISHED: 12 NODES REPORTING
          </div>
          <div className="log-entry">
            <span className="log-time">12:03:55</span>
            <span className="log-level warn">[WARN]</span> BATTERY_TEMP_THRESHOLD: NODE_04_P_WARM_02 (38°C)
          </div>
        </div>
      </div>

      {/* ── TELEMETRY SIDEBAR (RIGHT) ── */}
      <aside className="sim-sidebar">
        <div className="sidebar-title">TELEMETRY_MASTER</div>
        
        <div className="telemetry-card">
          <div className="card-label">ALTITUDE <span className="material-symbols-outlined">flight_takeoff</span></div>
          <div className="card-value">1,240.5 <span className="unit">M</span></div>
          <div className="progress-bar"><div className="fill" style={{ width: '65%' }}></div></div>
        </div>

        <div className="telemetry-card">
          <div className="card-label">VELOCITY <span className="material-symbols-outlined">speed</span></div>
          <div className="card-value">84.2 <span className="unit">KM/H</span></div>
          <div className="progress-bar"><div className="fill" style={{ width: '45%' }}></div></div>
        </div>

        <div className="telemetry-card system-mode">
          <div className="card-label">SYSTEM MODE <span className="status-dot"></span></div>
          <div className="card-value mode-text">AUTO_PILOT_SWARM</div>
          <div className="sub-value">SECTOR_GRID_B9</div>
        </div>

        <div className="telemetry-card coords">
          <div className="card-label">COORDINATES</div>
          <div className="coord-row"><span>LAT:</span> <span>36.1699° N</span></div>
          <div className="coord-row"><span>LNG:</span> <span>115.1398° W</span></div>
          <div className="coord-row"><span>HDG:</span> <span className="heading">284° NW</span></div>
        </div>

        <div className="telemetry-card">
          <div className="card-label">FLEET ENERGY <span className="material-symbols-outlined">battery_charging_full</span></div>
          <div className="card-value">78 <span className="unit">% AVG</span></div>
        </div>

        <div className="sidebar-footer">
          <div className="sim-time-label">SIM_TIME</div>
          <div className="sim-time-value">{formatTime(time)}</div>
          <button className="exit-btn" onClick={exitSim}>
            <span className="material-symbols-outlined">logout</span> EXIT
          </button>
        </div>
      </aside>
    </div>
  );
};

export { SimulationView };
