import { EventsOn } from "../../wailsjs/runtime/runtime";
import { lerpAngle } from "../utils/lerp-angle";
import { create } from "zustand";

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

interface State {
  interpolatedUavs: Record<string, TelemetryData>;
  subscribe(): void;
  unsuscribe(): void;
  setOnNewRealPoint(
    fn: (uavId: string, lat: number, lon: number, alt: number) => void,
  ): void;
}

export const useTelemetry = create<State>((set) => {
  const nodes: Record<string, InterpolationNode> = {};
  const lastPacketTimes: Record<string, number> = {};
  let animationFrameId: number | null = null;
  let unsuscribeTelemetry: (() => void) | undefined;
  let subscribed = false;
  let onNewRealPoint:
    | ((uavId: string, lat: number, lon: number, alt: number) => void)
    | undefined;

  const updatePositions = () => {
    const now = performance.now();
    const nextStates: Record<string, TelemetryData> = {};

    Object.entries(nodes).forEach(([id, node]) => {
      let t = (now - node.startTime) / node.duration;
      if (t > 1) t = 1; // Cap at 1 if we are waiting for the next packet

      const { start, end } = node;
      const iPos = start.payload.position;
      const fPos = end.payload.position;

      nextStates[id] = {
        ...end,
        payload: {
          ...end.payload,
          position: {
            lat: iPos.lat + (fPos.lat - iPos.lat) * t,
            lon: iPos.lon + (fPos.lon - iPos.lon) * t,
            alt: iPos.alt + (fPos.alt - iPos.alt) * t,
            relative_alt:
              iPos.relative_alt + (fPos.relative_alt - iPos.relative_alt) * t,
            heading: lerpAngle(iPos.heading, fPos.heading, t),
          },
        },
      };
    });

    set({ interpolatedUavs: nextStates });
    animationFrameId = requestAnimationFrame(updatePositions);
  };

  return {
    interpolatedUavs: {},

    setOnNewRealPoint(fn) {
      onNewRealPoint = fn;
    },

    subscribe() {
      if (subscribed) return;
      subscribed = true;

      unsuscribeTelemetry = EventsOn("telemetry", (data: TelemetryData) => {
        const now = performance.now();
        const uavId = data.uav_id;
        const pos = data.payload.position;
        data.last_update = now;

        onNewRealPoint?.(uavId, pos.lat, pos.lon, pos.alt);

        const lastNode = nodes[uavId];
        const lastPacketTime = lastPacketTimes[uavId];
        const duration = lastPacketTime ? now - lastPacketTime : 1000;
        lastPacketTimes[uavId] = now;

        if (data.payload.nr_gps_online > 0) {
          nodes[uavId] = {
            start: lastNode?.end ?? data,
            end: data,
            startTime: now,
            duration: duration,
          };
        }

        animationFrameId = requestAnimationFrame(updatePositions);
      });
    },

    unsuscribe() {
      unsuscribeTelemetry?.();
      unsuscribeTelemetry = undefined;
      if (animationFrameId) cancelAnimationFrame(animationFrameId);

      for (const key in nodes) delete nodes[key];
      for (const key in lastPacketTimes) delete lastPacketTimes[key];
      set({ interpolatedUavs: {} });
      subscribed = false;
    },
  };
});
