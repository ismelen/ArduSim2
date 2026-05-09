import type { InterpolationNode, TelemetryData } from "../hooks/useTelemetry";
import { lerpAngle } from "../utils/lerp-angle";

const nodes: Record<string, InterpolationNode> = {};
const lastPacketTimes: Record<string, number> = {};

const requestFrame =
  typeof self.requestAnimationFrame === "function"
    ? self.requestAnimationFrame.bind(self)
    : (callback: FrameRequestCallback) => setTimeout(callback, 16);

self.onmessage = (e) => {
  if (e.data.type === "NEW_DATA") {
    const { uavId, data, now, duration } = e.data;
    const lastpacketTime = lastPacketTimes[uavId];
    const newDuration = lastpacketTime ? now - lastpacketTime : duration;
    lastPacketTimes[uavId] = now;

    nodes[uavId] = {
      start: nodes[uavId]?.end ?? data,
      end: data,
      startTime: now,
      duration: newDuration,
    };
  }
};

const update = () => {
  const now = performance.now();
  const interpolated: Record<string, TelemetryData> = {};

  for (const id in nodes) {
    const node = nodes[id];
    let t = (now - node.startTime) / node.duration;
    if (t > 1) t = 1;

    const iPos = node.start.payload.position;
    const fPos = node.end.payload.position;

    interpolated[id] = {
      ...node.end,
      payload: {
        ...node.end.payload,
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
  }

  self.postMessage({ type: "TICK", interpolated });
  requestFrame(update);
};

update();
