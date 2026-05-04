import { useEffect, useRef } from "react";
import SimulationControls from "./components/simulation-controls";
import "maplibre-gl/dist/maplibre-gl.css";
import { useMap } from "../../hooks2/useMap";
import MapControls from "./components/map-controls";
import LogDisplay from "./components/log-display";

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
        <SimulationControls className="absolute top-2 left-2 z-50" />
        <MapControls className="absolute top-16 left-2 z-50" />
        <LogDisplay className="absolute bottom-2 left-2 right-2" />
      </div>
      <div
        className="h-full bg-white w-1/3 max-w-70 border-l 
      border-border shadow-sm overflow-y-auto"
      ></div>
    </main>
  );
}
