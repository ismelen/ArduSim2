import { useEffect, useRef, useState } from "react";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { lerpAngle } from "../utils2/lerp-angle";

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
  last_update: number;
}

interface InterpolationNode {
  start: TelemetryData;
  end: TelemetryData;
  startTime: number;
  duration: number;
}

export function useTelemetry(
  onNewRealPoint?: (uavId: string, lat: number, lon: number) => void,
) {
  const nodesRef = useRef<Record<string, InterpolationNode>>({});
  const lastPacketTimeRef = useRef<Record<string, number>>({});
  const [interpolatedUavs, setInterpolatedUavs] = useState<
    Record<string, TelemetryData>
  >({});
  const onNewRealPointRef = useRef(onNewRealPoint);

  // Keep the callback ref updated to avoid re-subscribing to events if it changes
  useEffect(() => {
    onNewRealPointRef.current = onNewRealPoint;
  }, [onNewRealPoint]);

  useEffect(() => {
    const unsubscribe = EventsOn("telemetry", (data: TelemetryData) => {
      const now = performance.now();
      const uavId = data.uav_id;
      data.last_update = now;

      // Notify about the new "point of truth" for the trail
      if (onNewRealPointRef.current) {
        onNewRealPointRef.current(
          uavId,
          data.payload.position.lat,
          data.payload.position.lon,
        );
      }

      const lastNode = nodesRef.current[uavId];
      const lastPacketTime = lastPacketTimeRef.current[uavId];

      // Estimate duration between packets for the next interpolation segment
      const duration = lastPacketTime ? now - lastPacketTime : 1000; // Default to 1s if first packet
      lastPacketTimeRef.current[uavId] = now;

      const isFirstValidPacket =
        (!lastNode ||
          (lastNode.end.payload.position.lat === 0 &&
            lastNode.end.payload.position.lon === 0)) &&
        (data.payload.position.lat !== 0 || data.payload.position.lon !== 0);

      if (isFirstValidPacket || !lastNode) {
        // First packet OR first non-zero packet: jump immediately (don't interpolate from 0,0)
        nodesRef.current[uavId] = {
          start: data,
          end: data,
          startTime: now,
          duration: 100, // Minimal duration for the very first appearance
        };
      } else {
        // We interpolate from the current end point (the "now" of the simulation) to the new point
        nodesRef.current[uavId] = {
          start: lastNode.end,
          end: data,
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
      const nextStates: Record<string, TelemetryData> = {};

      Object.entries(nodesRef.current).forEach(([id, node]) => {
        let t = (now - node.startTime) / node.duration;
        if (t > 1) t = 1; // Cap at 1 if we are waiting for the next packet

        const { start, end } = node;

        nextStates[id] = {
          ...end,
          payload: {
            ...end.payload,
            position: {
              lat:
                start.payload.position.lat +
                (end.payload.position.lat - start.payload.position.lat) * t,
              lon:
                start.payload.position.lon +
                (end.payload.position.lon - start.payload.position.lon) * t,
              alt:
                start.payload.position.alt +
                (end.payload.position.alt - start.payload.position.alt) * t,
              relative_alt:
                start.payload.position.relative_alt +
                (end.payload.position.relative_alt -
                  start.payload.position.relative_alt) *
                  t,
              heading: lerpAngle(
                start.payload.position.heading,
                end.payload.position.heading,
                t,
              ),
            },
          },
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
