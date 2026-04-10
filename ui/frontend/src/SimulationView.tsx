import React, { useEffect, useRef, useState } from 'react';
import './MapLibre.css';
import './SimulationView.css';
import { useConfig } from './hooks/useConfig';
import { useFleet } from './hooks/useFleet';
import type { UAVState } from './hooks/useTelemetry';
import { useTelemetry } from './hooks/useTelemetry';

// MapLibre imports
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import { EventsOff, EventsOn } from '../wailsjs/runtime/runtime';

interface NetSimMessageEvent {
  source: string;
  service: string;
  command: string;
  label: string;
}


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
  const { handleExitSimulation: exitSim, handleSendAlgorithmCommand } = useConfig();
  const { uavs: fleetUavs } = useFleet();
  const uavs = useTelemetry();
  const [time, setTime] = useState(0);
  const [viewMode, setViewMode] = useState<'2d' | '3d'>('2d');
  const [logs, setLogs] = useState<{ time: string, level: string, msg: string }[]>([]);
  const [showTrails, setShowTrails] = useState(true);
  const [simulationFinished, setSimulationFinished] = useState(false);
  // Tracks which algorithms are currently paused (true = paused, false = running).
  const [pausedAlgorithms, setPausedAlgorithms] = useState<Record<string, boolean>>({});
  const [startedAlgorithms, setStartedAlgorithms] = useState<Record<string, boolean>>({});

  const mapContainerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const markersRef = useRef<Record<string, maplibregl.Marker>>({});
  const trailsRef = useRef<Record<string, [number, number][]>>({});
  const centeredRef = useRef(false);
  const logEndRef = useRef<HTMLDivElement>(null);

  // Get current UAVs as a list
  const uavList: UAVState[] = Object.values(uavs);
  const mainUav = uavList.length > 0 ? uavList[0] : null;

  const runningAlgorithms = new Set<string>();
  fleetUavs.forEach(u => u.services?.forEach(s => runningAlgorithms.add(s.serviceId)));
  const algorithmsList = Array.from(runningAlgorithms);

  // Logic for enabling START mission button: all configured UAVs must have reported telemetry with at least one GPS satellite
  const allUavsReady = fleetUavs.length > 0 && fleetUavs.every(fu => uavs[fu.id] && uavs[fu.id].nrGpsOnline > 0);

  // Initialize MapLibre
  useEffect(() => {
    if (!mapContainerRef.current) return;

    const map = new maplibregl.Map({
      container: mapContainerRef.current,
      style: 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json',
      center: [0, 0],
      zoom: 13,
      pitch: viewMode === '3d' ? 60 : 0,
      maxPitch: 0,
      dragRotate: true,
      scrollZoom: true,
      dragPan: true,
      touchZoomRotate: true
    });


    map.on('load', () => {
      // Add GeoJSON trace layer
      map.addSource('uav-trails', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] }
      });
      map.addLayer({
        id: 'uav-trails-layer',
        type: 'line',
        source: 'uav-trails',
        layout: {
          'line-join': 'round',
          'line-cap': 'round',
          'visibility': showTrails ? 'visible' : 'none'
        },
        paint: {
          'line-color': '#4edea3',
          'line-width': 2,
          'line-opacity': 0.6
        }
      });

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
      map.remove();
      mapRef.current = null;
    };
  }, []);

  const handle2DSelect = () => {
    setViewMode('2d');
    if (mapRef.current) {
      mapRef.current.setMaxPitch(0);
      mapRef.current.setMinPitch(0);
      mapRef.current.easeTo({ pitch: 0, bearing: 0, duration: 1000 });
    }
  };

  const handle3DSelect = () => {
    setViewMode('3d');
    if (mapRef.current) {
      mapRef.current.setMaxPitch(85);
      mapRef.current.setMinPitch(0);
      mapRef.current.easeTo({ pitch: 60, bearing: 0, duration: 1000 });
    }
  };

  useEffect(() => {
    if (mapRef.current && mapRef.current.getLayer('uav-trails-layer')) {
      mapRef.current.setLayoutProperty('uav-trails-layer', 'visibility', showTrails ? 'visible' : 'none');
    }
  }, [showTrails]);

  // Update Markers and Auto-Center
  useEffect(() => {
    if (!mapRef.current) return;

    let featuresHasUpdates = false;

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

      // 3. Manage Trails
      if (!trailsRef.current[uav.id]) {
        trailsRef.current[uav.id] = [];
      }
      
      const lastPos = trailsRef.current[uav.id][trailsRef.current[uav.id].length - 1];
      // Only push if coordinate has actually moved and is valid (not 0,0)
      if ((!lastPos || lastPos[0] !== uav.lon || lastPos[1] !== uav.lat) && (uav.lon !== 0 || uav.lat !== 0)) {
        trailsRef.current[uav.id].push([uav.lon, uav.lat]);
        featuresHasUpdates = true;
      }
    });

    // Update GeoJSON source efficiently
    if (featuresHasUpdates) {
      const source = mapRef.current.getSource('uav-trails') as maplibregl.GeoJSONSource;
      if (source) {
        const features = Object.entries(trailsRef.current).map(([id, coords]) => ({
          type: 'Feature' as const,
          properties: { uavId: id },
          geometry: {
            type: 'LineString' as const,
            coordinates: coords
          }
        }));
        source.setData({ type: 'FeatureCollection', features });
      }
    }

  }, [uavs]);

  // Simulation timer
  useEffect(() => {
    if (simulationFinished) return;
    const interval = setInterval(() => setTime(t => t + 1), 1000);
    return () => clearInterval(interval);
  }, [simulationFinished]);

  // Log listener and netsim message listener
  useEffect(() => {
    // Docker-compose build / run logs from backend
    EventsOn('simulation:log', (message: string) => {
      const now = new Date();
      const timeStr = now.toLocaleTimeString([], { hour12: false });
      setLogs(prev => [...prev.slice(-49), { time: timeStr, level: '[INF]', msg: message }]);
    });

    // Messages from netsim (e.g. algorithm commands broadcast over the network)
    EventsOn('netsim:message', (data: NetSimMessageEvent) => {
      const now = new Date();
      const timeStr = now.toLocaleTimeString([], { hour12: false });
      console.log(`[netsim/messages] ${data.label}`);
      setLogs(prev => [...prev.slice(-49), { time: timeStr, level: '[MSG]', msg: data.label }]);
    });

    // Simulation finished signal: show banner, do NOT close the window.
    EventsOn('simulation:finished', () => {
      console.log('[netsim] simulation:finished received');
      setSimulationFinished(true);
    });

    return () => {
      EventsOff('simulation:log');
      EventsOff('netsim:message');
      EventsOff('simulation:finished');
    };
  }, []);

  // Log auto-scroll
  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [logs]);


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
      {/* ── SIMULATION FINISHED BANNER ── */}
      {simulationFinished && (
        <div className="sim-finished-banner">
          <span className="material-symbols-outlined">check_circle</span>
          <span>SIMULATION COMPLETE — all algorithms stopped</span>
        </div>
      )}

      {/* ── TOP CONTROLS ── */}
      <div className="sim-top-bar">
      {algorithmsList.map(algo => {
          const isStarted = !!startedAlgorithms[algo];
          const isPaused = !!pausedAlgorithms[algo];
          
          const handlePauseToggle = async () => {
            const command = isPaused ? 'resume' : 'pause';
            await handleSendAlgorithmCommand(algo, command);
            setPausedAlgorithms(prev => ({ ...prev, [algo]: !isPaused }));
          };

          const handleStart = async () => {
            await handleSendAlgorithmCommand(algo, 'start');
            setStartedAlgorithms(prev => ({ ...prev, [algo]: true }));
          };

          const handleStop = async () => {
             await handleSendAlgorithmCommand(algo, 'stop');
          };

          return (
          <div key={algo} className="control-group">
            <span className="algo-label">{algo.toUpperCase()}</span>
            <button 
              className={`control-btn outline-btn ${allUavsReady && !isStarted ? 'ready' : 'not-ready'}`}
              disabled={!allUavsReady || isStarted}
              onClick={handleStart}
            >
              <span className="material-symbols-outlined">play_circle</span> START
            </button>
            <button
              className={`control-btn ${isPaused ? 'outline-btn ready' : 'outline-btn'} ${!isStarted ? 'not-ready' : ''}`}
              disabled={!isStarted}
              onClick={handlePauseToggle}
              title={isPaused ? 'Resume algorithm' : 'Pause algorithm'}
            >
              <span className="material-symbols-outlined">{isPaused ? 'play_arrow' : 'pause'}</span>
              {isPaused ? 'RESUME' : 'PAUSE'}
            </button>
            <button 
              className={`control-btn ${isStarted ? 'danger-outline' : 'not-ready'}`} 
              disabled={!isStarted}
              onClick={handleStop}
            >
              <span className="material-symbols-outlined">stop</span> STOP
            </button>
          </div>
        );
        })}

        {!algorithmsList.length && (
          <div className="control-group">
            <button className="control-btn outline-btn disabled">
              NO ALGORITHMS
            </button>
          </div>
        )}
        
        <div className="view-toggle">
          <button className="toggle-btn active" onClick={handleRecenter}>
            <span className="material-symbols-outlined">center_focus_strong</span>
          </button>
        </div>
        <div className="view-toggle">
          <button 
            className={`toggle-btn ${showTrails ? 'active' : ''}`}
            onClick={() => setShowTrails(!showTrails)}
            title="Toggle Trails"
          >
            <span className="material-symbols-outlined">route</span>
          </button>
        </div>
        <div className="view-toggle">
          <button 
            className={`toggle-btn ${viewMode === '2d' ? 'active' : ''}`}
            onClick={handle2DSelect}
          >
            2D
          </button>
          <button 
            className={`toggle-btn ${viewMode === '3d' ? 'active' : ''}`}
            onClick={handle3DSelect}
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
            logs.map((log, i) => {
              const levelClass = log.level === '[MSG]' ? 'msg' : 'info';
              return (
                <div key={i} className="log-entry">
                  <span className="log-time">{log.time}</span>
                  <span className={`log-level ${levelClass}`}>{log.level}</span> {log.msg}
                </div>
              );
            })
          )}
          <div ref={logEndRef} />
        </div>

      </div>

      {/* ── TELEMETRY SIDEBAR (RIGHT) ── */}
      <aside className="sim-sidebar">
        <div className="sidebar-title">FLEET TELEMETRY</div>
        
        <div className="uav-cards-container">
          {uavList.length === 0 && (
            <div className="telemetry-card system-mode">
              <div className="card-value mode-text">AWAITING UAVs</div>
            </div>
          )}
          {uavList.map(uav => (
            <div key={uav.id} className="telemetry-card compact-card">
              <div className="card-label">UAV {uav.id} <span className={`status-dot ${uav.nrGpsOnline > 0 ? 'ready' : 'not-ready'}`}></span></div>
              
              <div className="compact-stats">
                <div className="stat">
                  <span className="material-symbols-outlined">flight_takeoff</span>
                  <span>{uav.alt.toFixed(1)} <small>m</small></span>
                </div>
                <div className="stat">
                  <span className="material-symbols-outlined">speed</span>
                  <span>{Math.sqrt(uav.vx**2 + uav.vy**2).toFixed(1)} <small>m/s</small></span>
                </div>
                <div className="stat">
                  <span className="material-symbols-outlined">battery_charging_full</span>
                  <span>{uav.battery}%</span>
                </div>
                <div className="stat">
                  <span className="mode-text">{uav.flight_mode}</span>
                </div>
              </div>

              <div className="compact-coords" style={{ fontFamily: 'var(--font-display)', fontSize: '0.65rem', color: 'var(--on-surface-variant)', marginTop: '0.4rem', letterSpacing: '0.05em' }}>
                <span style={{color: 'var(--on-surface)'}}>LAT</span> {uav.lat.toFixed(5)}° &bull; <span style={{color: 'var(--on-surface)'}}>LNG</span> {uav.lon.toFixed(5)}° &bull; <span style={{color: 'var(--primary)'}}>HDG</span> {uav.heading.toFixed(0)}°
              </div>
              
              <div className="progress-bar mini-bar">
                <div className="fill" style={{ width: `${uav.battery}%` }}></div>
              </div>
            </div>
          ))}
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
