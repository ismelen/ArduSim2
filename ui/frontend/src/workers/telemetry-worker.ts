import type { InterpolationNode, TelemetryData } from "../hooks/useTelemetry";
import { lerpAngle } from "../utils/lerp-angle";

const nodes: Record<string, InterpolationNode> = {};
let lastPacketTime: number | undefined;

const requestFrame =
  typeof self.requestAnimationFrame === "function"
    ? self.requestAnimationFrame.bind(self)
    : (callback: FrameRequestCallback) => setTimeout(callback, 16);

self.onmessage = (e) => {
  if (e.data.type === "NEW_SNAPSHOT") {
    const { uavs, now } = e.data;

    for (const [uavId, data] of Object.entries(uavs) as [
      string,
      TelemetryData,
    ][]) {
      const newDuration = lastPacketTime ? now - lastPacketTime : 1000;
      lastPacketTime = now;
      data.uav_id = uavId;  
      
      const savedData = nodes[uavId]

      nodes[uavId] = {
        start: savedData?.end ?? data,
        end: data,
        startTime: now,
        duration: newDuration,
        trailEmited: !savedData,
      };
    }
  }
};

const update = () => {
  const now = performance.now();
  const interpolated: Record<string, TelemetryData> = {};

  for (const id in nodes) {
    const node = nodes[id];
    let t = (now - node.startTime) / node.duration;
    if (t > 1) t = 1; //TODO: continue?

    const iPos = node.start.position;
    const fPos = node.end.position;

    if (!node.trailEmited) {
      node.trailEmited = true;
      self.postMessage({
        type: "POINT_REACHED",
        id,
        lat: iPos.lat,
        lon: iPos.lon,
        alt: iPos.alt,
      });
    }
    interpolated[id] = {
      ...node.end,
      position: {
        lat: iPos.lat + (fPos.lat - iPos.lat) * t,
        lon: iPos.lon + (fPos.lon - iPos.lon) * t,
        alt: iPos.alt + (fPos.alt - iPos.alt) * t,
        relative_alt:
          iPos.relative_alt + (fPos.relative_alt - iPos.relative_alt) * t,
        heading: lerpAngle(iPos.heading, fPos.heading, t),
      },
    };
  }

  self.postMessage({ type: "TICK", interpolated });
  requestFrame(update);
};

update();
