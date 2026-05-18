import { create } from "zustand";
import { EventsOn } from "../../wailsjs/runtime/runtime";

export interface TelemetryData {
  uav_id?: string;
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
}

export interface InterpolationNode {
  start: TelemetryData;
  end: TelemetryData;
  startTime: number;
  duration: number;
  trailEmited: boolean;
}

interface State {
  interpolatedUavs: () => Record<string, TelemetryData>;
  subscribe(): void;
  unsuscribe(): void;
  setOnNewRealPoint(
    fn: (uavId: string, lat: number, lon: number, alt: number) => void,
  ): void;
  reset(): void;
}

export const useTelemetry = create<State>((_, get) => {
  const rawData = { current: {} as Record<string, TelemetryData> };
  const worker = new Worker(
    new URL("../workers/telemetry-worker.ts", import.meta.url),
    { type: "module" },
  );

  worker.onmessage = (e) => {
    if (e.data.type === "TICK") {
      rawData.current = e.data.interpolated;
    }
    if (e.data.type == "POINT_REACHED") {
      onNewRealPoint?.(e.data.id, e.data.lat, e.data.lon, e.data.alt);
    }
  };

  let unsuscribeTelemetry: (() => void) | undefined;
  let subscribed = false;
  let onNewRealPoint:
    | ((uavId: string, lat: number, lon: number, alt: number) => void)
    | undefined;

  return {
    interpolatedUavs: () => rawData.current,

    setOnNewRealPoint(fn) {
      onNewRealPoint = fn;
    },

    subscribe() {
      if (subscribed) return;
      subscribed = true;

      unsuscribeTelemetry = EventsOn(
        "telemetry_snapshot",
        (payload: { uavs: Record<string, TelemetryData> }) => {
          worker.postMessage({
            type: "NEW_SNAPSHOT",
            uavs: payload.uavs,
            now: performance.now(),
          });
        },
      );
    },

    unsuscribe() {
      unsuscribeTelemetry?.();
      unsuscribeTelemetry = undefined;
      rawData.current = {};
      subscribed = false;
      worker.terminate();
    },

    reset() {
      get().unsuscribe();
    },
  };
});
