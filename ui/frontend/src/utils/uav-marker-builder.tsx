import { getUavColor } from "../constants/uav-colors";

export function createUAVMarkerElement(
  heading: number,
  altitude: number,
  idx: number,
) {
  const el = document.createElement("div");
  el.className = "uav-marker";
  el.style.setProperty("--uav-color", getUavColor(idx));
  el.innerHTML = `
    <div class="uav-sphere-container">
      <div class="uav-sphere" style="transform: rotate(${heading}deg);"></div>
    </div>
    <div class="uav-altitude-tag">${altitude.toFixed(1)}m</div>
  `;
  return el;
}
