import React, { useEffect, useRef, useState } from 'react';
import './MapLibre.css';
import './SimulationView.css';
import { useConfig } from './hooks/useConfig';
import type { UAVState } from './hooks/useTelemetry';
import { useTelemetry } from './hooks/useTelemetry';

// MapLibre imports
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import { EventsOff, EventsOn } from '../wailsjs/runtime/runtime';


// Custom UAV Marker HTML generator
const createUAVMarkerElement = (heading: number, altitude: number) => {
  const el = document.createElement('div');
  el.className = 'uav-marker';
  el.innerHTML = `
    <div class="uav-sphere-container">
      <div class="uav-sphere" style="transform: rotate(${heading}deg);"></div>
    </div>
    <div class="uav-altitude-tag">${altitude.toFixed(1)}m</div>
  `;
  return el;
};

const SimulationView: React.FC = () => {
  const { handleExitSimulation: exitSim } = useConfig();
  const uavs = useTelemetry();
  const [time, setTime] = useState(0);
  const [viewMode, setViewMode] = useState<'2d' | '3d'>('2d');
  const [logs, setLogs] = useState<{ time: string, level: string, msg: string }[]>([]);

  
  const mapContainerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const markersRef = useRef<Record<string, maplibregl.Marker>>({});
  const centeredRef = useRef(false);

  // Get current UAVs as a list
  const uavList: UAVState[] = Object.values(uavs);
  const mainUav = uavList.length > 0 ? uavList[0] : null;

  // Initialize MapLibre
  useEffect(() => {
    if (!mapContainerRef.current) return;

    const map = new maplibregl.Map({
      container: mapContainerRef.current,
      style: 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json',
      center: [0, 0],
      zoom: 13,
      pitch: viewMode === '3d' ? 60 : 0,
      dragRotate: true,
      scrollZoom: true,
      dragPan: true,
      touchZoomRotate: true
    });


    // ── NATIVE INTERACTION Logic for Middle-click (button 1) ──
    const container = mapContainerRef.current;
    let isRotating = false;

    const handleMouseDown = (e: MouseEvent) => {
      // ONLY Right-click (2) to rotate/pitch. Middle-click (1) removed as requested.
      if (e.button === 2) {
        e.preventDefault();
        isRotating = true;
        if (container) container.style.cursor = 'grabbing';
      }
    };

    const handleMouseMove = (e: MouseEvent) => {
      if (isRotating && mapRef.current) {
        const deltaX = e.movementX;
        const deltaY = e.movementY;
        
        // Much lower sensitivity for ultra-precise rotation (0.08)
        const bearing = mapRef.current.getBearing() + deltaX * 0.005;
        const pitch = Math.max(0, Math.min(85, mapRef.current.getPitch() - deltaY * 0.005));
        
        mapRef.current.setBearing(bearing);
        mapRef.current.setPitch(pitch);
      }
    };

    const handleMouseUp = () => {
      isRotating = false;
      if (container) container.style.cursor = '';
    };

    const handleContextMenu = (e: MouseEvent) => {
      e.preventDefault();
    };

    container.addEventListener('mousedown', handleMouseDown);
    container.addEventListener('contextmenu', handleContextMenu);
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);

    map.on('load', () => {
      // Add 3D Building Layer (fill-extrusion)
      const layers = map.getStyle().layers;
      const labelLayerId = layers?.find(l => l.type === 'symbol' && l.layout && l.layout['text-field'])?.id;

      map.addLayer({
        'id': '3d-buildings',
        'source': 'carto',
        'source-layer': 'building',
        'type': 'fill-extrusion',
        'minzoom': 15,
        'paint': {
          'fill-extrusion-color': '#4edea3',
          'fill-extrusion-height': ['get', 'render_height'],
          'fill-extrusion-base': ['get', 'render_min_height'],
          'fill-extrusion-opacity': 0.6
        }
      }, labelLayerId);
    });

    map.addControl(new maplibregl.NavigationControl({ showCompass: true }), 'top-right');
    mapRef.current = map;

    return () => {
      container.removeEventListener('mousedown', handleMouseDown);
      container.removeEventListener('contextmenu', handleContextMenu);
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      map.remove();
      mapRef.current = null;
    };
  }, []);

  // Handle View Mode changes
  useEffect(() => {
    if (mapRef.current) {
      mapRef.current.easeTo({
        pitch: viewMode === '3d' ? 60 : 0,
        duration: 1000
      });
    }
  }, [viewMode]);

  // Update Markers and Auto-Center
  useEffect(() => {
    if (!mapRef.current) return;

    uavList.forEach((uav) => {
      // 1. Auto-center on first valid coordinate (NOT 0,0)
      if (!centeredRef.current && (uav.lat !== 0 || uav.lon !== 0)) {
        mapRef.current?.flyTo({
          center: [uav.lon, uav.lat],
          zoom: 16,
          speed: 1.2
        });
        centeredRef.current = true;
      }

      // 2. Manage Markers
      if (markersRef.current[uav.id]) {
        // Update existing marker
        const marker = markersRef.current[uav.id];
        marker.setLngLat([uav.lon, uav.lat]);
        
        // Update marker rotation and altitude label
        const el = marker.getElement();
        const sphereEl = el.querySelector('.uav-sphere') as HTMLElement;
        const tagEl = el.querySelector('.uav-altitude-tag') as HTMLElement;
        if (sphereEl) sphereEl.style.transform = `rotate(${uav.heading}deg)`;
        if (tagEl) tagEl.textContent = `${uav.alt.toFixed(1)}m`;
      } else {
        // Create new marker
        const el = createUAVMarkerElement(uav.heading, uav.alt);
        const marker = new maplibregl.Marker({ element: el })
          .setLngLat([uav.lon, uav.lat])
          .addTo(mapRef.current!);
        markersRef.current[uav.id] = marker;
      }
    });

    // Cleanup markers for offline UAVs (optional, depending on requirements)
  }, [uavs]);

  // Simulation timer and Log listener
  useEffect(() => {
    const interval = setInterval(() => setTime(t => t + 1), 1000);
    
    // Listen to real-time logs from backend
    EventsOn('simulation:log', (message: string) => {
      const now = new Date();
      const timeStr = now.toLocaleTimeString([], { hour12: false });
      setLogs(prev => [...prev.slice(-49), { time: timeStr, level: '[INF]', msg: message }]);
    });

    return () => {
      clearInterval(interval);
      EventsOff('simulation:log');
    };
  }, []);


  const formatTime = (seconds: number) => {
    const h = Math.floor(seconds / 3600).toString().padStart(2, '0');
    const m = Math.floor((seconds % 3600) / 60).toString().padStart(2, '0');
    const s = (seconds % 60).toString().padStart(2, '0');
    return `${h}:${m}:${s}`;
  };

  const handleRecenter = () => {
    if (mapRef.current && mainUav) {
      mapRef.current.flyTo({
        center: [mainUav.lon, mainUav.lat],
        zoom: 17,
        speed: 1.5
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
          <button 
            className={`toggle-btn ${viewMode === '2d' ? 'active' : ''}`}
            onClick={() => setViewMode('2d')}
          >
            2D
          </button>
          <button 
            className={`toggle-btn ${viewMode === '3d' ? 'active' : ''}`}
            onClick={() => setViewMode('3d')}
          >
            3D
          </button>
        </div>
      </div>

      {/* ── MAP AREA (CENTER) ── */}
      <div className="sim-map-area">
        <div ref={mapContainerRef} style={{ height: '100%', width: '100%' }} />
        <div className="map-grid-overlay"></div>
      </div>

      {/* ── MISSION LOG (BOTTOM LEFT) ── */}
      <div className="sim-log-panel">
        <div className="panel-header">
          <span className="material-symbols-outlined log-header-icon">terminal</span> 
          <span className="log-header-title">LOG</span>
        </div>
        <div className="log-content">
          {logs.length === 0 ? (
            <div className="log-entry">
              <span className="log-time">--:--:--</span>
              <span className="log-level info">[WAIT]</span> AWAITING_SIMULATION_UPLINK...
            </div>
          ) : (
            logs.map((log, i) => (
              <div key={i} className="log-entry">
                <span className="log-time">{log.time}</span>
                <span className="log-level info">{log.level}</span> {log.msg}
              </div>
            ))
          )}
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
            <div className="fill" style={{ width: `${Math.min((mainUav?.alt ?? 0) / 20, 100)}%` }}></div>
          </div>
        </div>

        <div className="telemetry-card">
          <div className="card-label">VELOCITY <span className="material-symbols-outlined">speed</span></div>
          <div className="card-value">
            {mainUav ? Math.sqrt(mainUav.vx**2 + mainUav.vy**2).toFixed(1) : '0.0'} 
            <span className="unit">M/S</span>
          </div>
          <div className="progress-bar">
            <div className="fill" style={{ width: `${Math.min(Math.sqrt((mainUav?.vx ?? 0)**2 + (mainUav?.vy ?? 0)**2) * 5, 100)}%` }}></div>
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
