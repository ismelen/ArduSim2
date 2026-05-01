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
const createUAVMarkerElement = (heading: number, altitude: number, color: string) => {
  const el = document.createElement('div');
  el.className = 'uav-marker';
  el.style.setProperty('--uav-color', color);
  el.innerHTML = `
    <div class="uav-sphere-container">
      <div class="uav-sphere" style="transform: rotate(${heading}deg);"></div>
    </div>
    <div class="uav-altitude-tag">${altitude.toFixed(1)}m</div>
  `;
  return el;
};

interface SplitButtonProps {
  label: string;
  icon: string;
  options: string[];
  disabled: boolean;
  activeClass?: string;
  onMainClick: () => void;
  onOptionClick: (option: string) => void;
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
}

const SplitButton: React.FC<SplitButtonProps> = ({
  label, icon, options, disabled, activeClass = '', onMainClick, onOptionClick, isOpen, setIsOpen
}) => {
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen, setIsOpen]);

  return (
    <div className={`split-btn-group ${disabled ? 'disabled' : ''} ${activeClass}`} ref={dropdownRef}>
      <button 
        className="split-btn-main"
        onClick={onMainClick}
        disabled={disabled}
      >
        <span className="material-symbols-outlined">{icon}</span> {label}
      </button>
      <button  
        className="split-btn-arrow"
        onClick={() => setIsOpen(!isOpen)}
        disabled={disabled || options.length === 0}
      >
        <span className="material-symbols-outlined" style={{ fontSize: '1rem' }}>{isOpen ? 'expand_less' : 'expand_more'}</span>
      </button>
      
      {isOpen && options.length > 0 && (
        <div className="split-btn-dropdown">
          {options.map(opt => (
            <div key={opt} className="split-dropdown-item" onClick={() => {
              onOptionClick(opt);
              setIsOpen(false);
            }}>
              {opt}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

const SimulationView: React.FC = () => {
    const { handleExitSimulation: exitSim, handleSendAlgorithmCommand, isExiting } = useConfig();
  const { uavs: fleetUavs } = useFleet();
  const uavs = useTelemetry((uavId, lat, lon) => {
    if (!trailsRef.current[uavId]) {
      trailsRef.current[uavId] = [];
    }
    // Only push if it's a valid coordinate (not 0,0)
    if (lat !== 0 || lon !== 0) {
      const coords = trailsRef.current[uavId];
      const last = coords[coords.length - 1];
      // Push if it's the first point or different from the last real point to keep history clean
      if (!last || last[0] !== lon || last[1] !== lat) {
        coords.push([lon, lat]);
      }

      // Auto-center on first valid coordinate (NOT 0,0)
      if (!centeredRef.current && mapRef.current) {
        mapRef.current.flyTo({
          center: [lon, lat],
          zoom: 16,
          speed: 1.2
        });
        centeredRef.current = true;
      }
    }
  });
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
  const uavColors = useRef<Record<string, string>>({});
  const centeredRef = useRef(false);
  const logEndRef = useRef<HTMLDivElement>(null);
  const [openDropdown, setOpenDropdown] = useState<'start' | 'pause' | 'stop' | null>(null);

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
      style: 'https://basemaps.cartocdn.com/gl/voyager-gl-style/style.json',
      center: [0, 0],
      zoom: 13,
      pitch: viewMode === '3d' ? 60 : 0,
      maxPitch: 0,
      dragRotate: true,
      scrollZoom: true,
      dragPan: true,
      touchZoomRotate: true,
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
          'line-color': ['get', 'color'],
          'line-width': 4,
          'line-opacity': 0.8
        }
      });

      map.addLayer({
        'id': '3d-buildings',
        'source': 'carto',
        'source-layer': 'building',
        'type': 'fill-extrusion',
        'minzoom': 15,
        'paint': {
          'fill-extrusion-color': '#d1d5db',
          'fill-extrusion-height': ['coalesce', ['get', 'render_height'], ['get', 'height'], 15],
          'fill-extrusion-base': ['coalesce', ['get', 'render_min_height'], ['get', 'min_height'], 0],
          'fill-extrusion-opacity': .5,
          "fill-extrusion-vertical-gradient": true
        }
      });

      map.setLight({
        anchor: 'viewport',
        color: 'white',
        intensity: 0.3,
        position: [1.15, 90, 40] // [radial, azimuthal, polar]
      });
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

  const getColorForUav = (id: string) => {
    if (!uavColors.current[id]) {
      const colors = ['#ef4444', '#3b82f6', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316'];
      const index = Object.keys(uavColors.current).length % colors.length;
      uavColors.current[id] = colors[index];
    }
    return uavColors.current[id];
  };

  // Update Markers and Auto-Center
  useEffect(() => {
    if (!mapRef.current) return;

    uavList.forEach((uav) => {
      // 1. Manage Markers
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
        const el = createUAVMarkerElement(uav.heading, uav.alt, getColorForUav(uav.id));
        const marker = new maplibregl.Marker({ element: el })
          .setLngLat([uav.lon, uav.lat])
          .addTo(mapRef.current!);
        markersRef.current[uav.id] = marker;
      }
    });

    // 3. Manage Trails (GeoJSON layer)
    // Always update GeoJSON source every frame so the tip of the trail perfectly anchors to the UAV
    const source = mapRef.current.getSource('uav-trails') as maplibregl.GeoJSONSource;
    if (source) {
      const features = Object.entries(trailsRef.current).map(([id, coords]) => {
        const uavState = uavList.find(u => u.id === id);
        const activeCoords = [...coords];
        // Append current interpolated location to bridge the gap between discrete telemetry points
        if (uavState && uavState.lon !== 0 && uavState.lat !== 0) {
          activeCoords.push([uavState.lon, uavState.lat]);
        }
        
        return {
          type: 'Feature' as const,
          properties: { uavId: id, color: getColorForUav(id) },
          geometry: {
            type: 'LineString' as const,
            coordinates: activeCoords
          }
        };
      }).filter(f => f.geometry.coordinates.length >= 2);

      source.setData({ type: 'FeatureCollection', features });
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

  const allStarted = algorithmsList.every(algo => startedAlgorithms[algo]);
  const anyStarted = algorithmsList.some(algo => startedAlgorithms[algo]);
  const globalPaused = algorithmsList.some(algo => pausedAlgorithms[algo]);

  const handleStart = async (algo?: string) => {
    const targets = algo ? [algo] : algorithmsList;
    for (const a of targets) {
      if (!startedAlgorithms[a]) {
        await handleSendAlgorithmCommand(a, 'start');
        setStartedAlgorithms(prev => ({ ...prev, [a]: true }));
      }
    }
  };

  const handlePauseToggle = async (algo?: string) => {
    const targets = algo ? [algo] : algorithmsList;
    const isTargetPaused = algo ? pausedAlgorithms[algo] : globalPaused;
    const command = isTargetPaused ? 'resume' : 'pause';
    for (const a of targets) {
      await handleSendAlgorithmCommand(a, command);
      setPausedAlgorithms(prev => ({ ...prev, [a]: !isTargetPaused }));
    }
  };

  const handleStop = async (algo?: string) => {
    const targets = algo ? [algo] : algorithmsList;
    for (const a of targets) {
      await handleSendAlgorithmCommand(a, 'stop');
    }
  };

  return (
    <div className="sim-container">
      {/* ── SIMULATION EXITING OVERLAY ── */}
      {isExiting && (
        <div className="sim-exiting-overlay">
          <div className="exiting-content">
            <div className="exiting-spinner"></div>
            <div className="exiting-text-group">
              <div className="exiting-title">SHUTTING DOWN SIMULATION</div>
              <div className="exiting-subtitle">Removing containers and cleaning environment...</div>
            </div>
          </div>
        </div>
      )}

      {/* ── SIMULATION FINISHED BANNER ── */}
      {simulationFinished && (
        <div className="sim-finished-banner">
          <span className="material-symbols-outlined">check_circle</span>
          <span>SIMULATION COMPLETE — all algorithms stopped</span>
        </div>
      )}

      {/* ── TOP CONTROLS ── */}
      <div className="sim-top-bar">
        <div className="control-group">
          {algorithmsList.length > 0 ? (
            <>
              <SplitButton 
                label="START"
                icon="play_circle"
                options={algorithmsList.filter(a => !startedAlgorithms[a])}
                disabled={!allUavsReady || allStarted}
                activeClass={allUavsReady && !allStarted ? 'btn-solid-ready' : ''}
                onMainClick={() => handleStart()}
                onOptionClick={(opt) => handleStart(opt)}
                isOpen={openDropdown === 'start'}
                setIsOpen={(open) => setOpenDropdown(open ? 'start' : null)}
              />
              <SplitButton 
                label={globalPaused ? "RESUME" : "PAUSE"}
                icon={globalPaused ? "play_arrow" : "pause"}
                options={algorithmsList}
                disabled={!anyStarted}
                activeClass={globalPaused ? 'btn-outline-ready' : ''}
                onMainClick={() => handlePauseToggle()}
                onOptionClick={(opt) => handlePauseToggle(opt)}
                isOpen={openDropdown === 'pause'}
                setIsOpen={(open) => setOpenDropdown(open ? 'pause' : null)}
              />
              <SplitButton 
                label="STOP"
                icon="stop"
                options={algorithmsList.filter(a => startedAlgorithms[a])}
                disabled={!anyStarted}
                activeClass={anyStarted ? 'btn-danger' : ''}
                onMainClick={() => handleStop()}
                onOptionClick={(opt) => handleStop(opt)}
                isOpen={openDropdown === 'stop'}
                setIsOpen={(open) => setOpenDropdown(open ? 'stop' : null)}
              />
            </>
          ) : (
            <button className="control-btn outline-btn disabled">
              NO ALGORITHMS
            </button>
          )}
        </div>
        
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
          {uavList.map(uav => {
            const color = getColorForUav(uav.id);
            return (
              <div 
                key={uav.id} 
                className="telemetry-card compact-card"
                style={{ '--uav-color': color } as React.CSSProperties}
              >
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
                  <span style={{color: 'var(--on-surface)'}}>LAT</span> {uav.lat.toFixed(5)}° &bull; <span style={{color: 'var(--on-surface)'}}>LNG</span> {uav.lon.toFixed(5)}° &bull; <span style={{color: 'var(--uav-color)'}}>HDG</span> {uav.heading.toFixed(0)}°
                </div>
                
                <div className="progress-bar mini-bar">
                  <div className="fill" style={{ width: `${uav.battery}%` }}></div>
                </div>
              </div>
            );
          })}
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
