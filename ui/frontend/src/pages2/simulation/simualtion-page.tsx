import "maplibre-gl/dist/maplibre-gl.css";
import { useEffect, useRef } from "react";
import { useMap } from "../../hooks2/useMap";
import LogDisplay from "./components/log-display";
import MapControls from "./components/map-controls";
import SimulationControls from "./components/simulation-controls";

export default function SimulatinPage() {
  const mapContainerRef = useRef<HTMLDivElement>(null);
  const initMap = useMap((s) => s.init);
  const showTrails = useMap((s) => s.showTrails);

  useEffect(() => {
    if (!mapContainerRef.current) return;
    return initMap(mapContainerRef.current);
  }, [initMap, showTrails]);

  return (
    <main className="flex h-full">
      <div ref={mapContainerRef} className="h-full relative flex-1">
        <MapControls className="absolute top-16 left-2 z-50" />
        <SimulationControls className="absolute top-2 left-2 z-50" />
        <LogDisplay className="absolute bottom-2 left-2 right-2" />
      </div>
      <div
        className="h-full bg-cwhite w-1/3 max-w-70 border-l 
      border-border shadow-sm overflow-y-auto"
      ></div>
    </main>
  );
}
