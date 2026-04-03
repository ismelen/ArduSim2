import React, { useEffect, useState, useRef } from 'react';
import './SimulationView.css';
import { useConfig } from './hooks/useConfig';
import { useTelemetry } from './hooks/useTelemetry';

// Leaflet imports
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

// Custom UAV Icon (SVG-based for better aesthetics)
const createUAVIcon = (heading: number, color: string = '#4edea3') => {
  return L.divIcon({
    className: 'uav-marker-container',
    html: `
      <div class="uav-marker-wrapper" style="transform: rotate(${heading}deg);">
        <svg width="32" height="32" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M16 4L28 26L16 22L4 26L16 4Z" fill="${color}" stroke="white" stroke-width="1.5" stroke-linejoin="round"/>
          <circle cx="16" cy="16" r="3" fill="white" opacity="0.5"/>
        </svg>
      </div>
    `,
    iconSize: [32, 32],
    iconAnchor: [16, 16]
  });
};

// Component to auto-center map when first UAV appears
const AutoCenter: React.FC<{ pos: [number, number] | null }> = ({ pos }) => {
  const map = useMap();
  const centered = useRef(false);
  
  useEffect(() => {
    if (pos && !centered.current) {
      map.setView(pos, 16);
      centered.current = true;
    }
  }, [pos, map]);
  
  return null;
};

const SimulationView: React.FC = () => {
  const { handleExitSimulation: exitSim } = useConfig();
  const uavs = useTelemetry();
  const [time, setTime] = useState(0);
  const mapRef = useRef<L.Map | null>(null);

  // Get the first UAV as "Master" for the sidebar display
  const uavList = Object.values(uavs);
  const mainUav = uavList.length > 0 ? uavList[0] : null;
  const initialPos: [number, number] | null = mainUav ? [mainUav.lat, mainUav.lon] : null;

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

  const handleRecenter = () => {
    if (mapRef.current && initialPos) {
      mapRef.current.flyTo(initialPos, 18, {
        duration: 1.5
      });
    }
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
          <button className="toggle-btn active" onClick={handleRecenter}>
            <span className="material-symbols-outlined">center_focus_strong</span> RECENTER
          </button>
        </div>
        <div className="view-toggle">
          <button className="toggle-btn active">2D</button>
          <button className="toggle-btn">3D</button>
        </div>
      </div>

      {/* ── MAP AREA (CENTER) ── */}
      <div className="sim-map-area">
        <MapContainer 
          ref={mapRef}
          center={[36.1699, -115.1398]} 
          zoom={13} 
          scrollWheelZoom={true}
          style={{ height: '100%', width: '100%' }}
          zoomControl={false}
        >
          <TileLayer
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>'
            url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
          />
          
          <AutoCenter pos={initialPos} />

          {uavList.map((uav) => (
            <Marker 
              key={uav.id} 
              position={[uav.lat, uav.lon]} 
              icon={createUAVIcon(uav.heading)}
            >
              <Popup>
                <div className="uav-popup">
                  <strong>{uav.id}</strong><br/>
                  Alt: {uav.alt.toFixed(1)}m<br/>
                  Mode: {uav.flight_mode}
                </div>
              </Popup>
            </Marker>
          ))}
          
          <div className="map-grid-overlay"></div>
        </MapContainer>
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
          <div className="card-value">
            {mainUav ? mainUav.alt.toLocaleString(undefined, { minimumFractionDigits: 1 }) : '0.0'} 
            <span className="unit">M</span>
          </div>
          <div className="progress-bar">
            <div className="fill" style={{ width: `${Math.min((mainUav?.alt || 0) / 20, 100)}%` }}></div>
          </div>
        </div>

        <div className="telemetry-card">
          <div className="card-label">VELOCITY <span className="material-symbols-outlined">speed</span></div>
          <div className="card-value">
            {mainUav ? Math.sqrt(mainUav.vx**2 + mainUav.vy**2).toFixed(1) : '0.0'} 
            <span className="unit">M/S</span>
          </div>
          <div className="progress-bar">
            <div className="fill" style={{ width: `${Math.min(Math.sqrt((mainUav?.vx || 0)**2 + (mainUav?.vy || 0)**2) * 5, 100)}%` }}></div>
          </div>
        </div>

        <div className="telemetry-card system-mode">
          <div className="card-label">SYSTEM MODE <span className="status-dot"></span></div>
          <div className="card-value mode-text">{mainUav?.flight_mode || 'OFFLINE'}</div>
          <div className="sub-value">STATUS: {mainUav?.status || 'UNKNOWN'}</div>
        </div>

        <div className="telemetry-card coords">
          <div className="card-label">COORDINATES</div>
          <div className="coord-row"><span>LAT:</span> <span>{mainUav?.lat.toFixed(6) || '0.000000'}°</span></div>
          <div className="coord-row"><span>LNG:</span> <span>{mainUav?.lon.toFixed(6) || '0.000000'}°</span></div>
          <div className="coord-row"><span>HDG:</span> <span className="heading">{mainUav?.heading.toFixed(0) || '0'}°</span></div>
        </div>

        <div className="telemetry-card">
          <div className="card-label">FLEET ENERGY <span className="material-symbols-outlined">battery_charging_full</span></div>
          <div className="card-value">{mainUav?.battery || 0} <span className="unit">% AVG</span></div>
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
