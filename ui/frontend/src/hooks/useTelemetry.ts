import { useEffect, useRef, useState } from "react";
import { EventsOn } from "../../wailsjs/runtime/runtime";

/**
 * METERS_PER_DEGREE_LAT: Approximate meters per degree of latitude.
 * 111,319.5m is a standard constant for WGS84 at most latitudes.
 */
// const METERS_PER_DEGREE_LAT = 111319.5;

export interface TelemetryData {
  uav_id: string;
  payload: {
    position: {
      lat: number;
      lon: number;
      alt: number;
      heading: number;
      relative_alt: number;
    };
    speed: { vx: number; vy: number; vz: number };
    battery: number;
    status: string;
    flight_mode: string;
    nr_gps_online: number;
    time_boot_ms: number;
  };
}

export interface UAVState {
  id: string;
  lat: number;
  lon: number;
  alt: number;
  relativeAlt: number;
  heading: number;
  vx: number; // North (m/s)
  vy: number; // East (m/s)
  vz: number; // Down (m/s)
  lastUpdate: number;
  battery: number;
  status: string;
  flight_mode: string;
  nrGpsOnline: number;
}

interface InterpolationNode {
  start: UAVState;
  end: UAVState;
  startTime: number;
  duration: number;
}

/**
 * Helper to interpolate between two angles correctly (handles 359 -> 0 jump)
 */
function lerpAngle(a: number, b: number, t: number) {
  let diff = ((b - a + 180) % 360) - 180;
  return a + diff * t;
}

/**
 * useTelemetry hook manages real-time UAV telemetry state.
 * It uses interpolation between the last two received points to provide smooth 60fps movement.
 */
export function useTelemetry(
  onNewRealPoint?: (uavId: string, lat: number, lon: number) => void,
) {
  const nodesRef = useRef<Record<string, InterpolationNode>>({});
  const lastPacketTimeRef = useRef<Record<string, number>>({});
  const [interpolatedUavs, setInterpolatedUavs] = useState<
    Record<string, UAVState>
  >({});
  const onNewRealPointRef = useRef(onNewRealPoint);

  // Keep the callback ref updated to avoid re-subscribing to events if it changes
  useEffect(() => {
    onNewRealPointRef.current = onNewRealPoint;
  }, [onNewRealPoint]);

  useEffect(() => {
    const unsubscribe = EventsOn("telemetry", (msg: TelemetryData) => {
      const now = performance.now();
      const uavId = msg.uav_id;

      const newState: UAVState = {
        id: uavId,
        lat: msg.payload.position.lat,
        lon: msg.payload.position.lon,
        alt: msg.payload.position.alt,
        relativeAlt: msg.payload.position.relative_alt,
        heading: msg.payload.position.heading,
        vx: msg.payload.speed.vx,
        vy: msg.payload.speed.vy,
        vz: msg.payload.speed.vz,
        lastUpdate: now,
        battery: msg.payload.battery,
        status: msg.payload.status,
        flight_mode: msg.payload.flight_mode,
        nrGpsOnline: msg.payload.nr_gps_online,
      };

      // Notify about the new "point of truth" for the trail
      if (onNewRealPointRef.current) {
        onNewRealPointRef.current(uavId, newState.lat, newState.lon);
      }

      const lastNode = nodesRef.current[uavId];
      const lastPacketTime = lastPacketTimeRef.current[uavId];

      // Estimate duration between packets for the next interpolation segment
      const duration = lastPacketTime ? now - lastPacketTime : 1000; // Default to 1s if first packet
      lastPacketTimeRef.current[uavId] = now;

      const isFirstValidPacket =
        (!lastNode || (lastNode.end.lat === 0 && lastNode.end.lon === 0)) &&
        (newState.lat !== 0 || newState.lon !== 0);

      if (isFirstValidPacket || !lastNode) {
        // First packet OR first non-zero packet: jump immediately (don't interpolate from 0,0)
        nodesRef.current[uavId] = {
          start: newState,
          end: newState,
          startTime: now,
          duration: 100, // Minimal duration for the very first appearance
        };
      } else {
        // We interpolate from the current end point (the "now" of the simulation) to the new point
        nodesRef.current[uavId] = {
          start: lastNode.end,
          end: newState,
          startTime: now,
          duration: duration,
        };
      }
    });

    return () => {
      if (unsubscribe) unsubscribe();
    };
  }, []); // No dependencies, subscription handled via refs

  useEffect(() => {
    let animationFrameId: number;

    const updatePositions = () => {
      const now = performance.now();
      const nextStates: Record<string, UAVState> = {};

      Object.entries(nodesRef.current).forEach(([id, node]) => {
        let t = (now - node.startTime) / node.duration;
        if (t > 1) t = 1; // Cap at 1 if we are waiting for the next packet

        const { start, end } = node;

        nextStates[id] = {
          ...end, // Base state from the target node
          lat: start.lat + (end.lat - start.lat) * t,
          lon: start.lon + (end.lon - start.lon) * t,
          alt: start.alt + (end.alt - start.alt) * t,
          relativeAlt:
            start.relativeAlt + (end.relativeAlt - start.relativeAlt) * t,
          heading: lerpAngle(start.heading, end.heading, t),
          // Velocities and other stats are just kept from the 'end' node for simplicity
        };
      });

      setInterpolatedUavs(nextStates);
      animationFrameId = requestAnimationFrame(updatePositions);
    };

    animationFrameId = requestAnimationFrame(updatePositions);
    return () => cancelAnimationFrame(animationFrameId);
  }, []);

  return interpolatedUavs;
}
