import "maplibre-gl/dist/maplibre-gl.css";
import { useEffect, useRef } from "react";
import { useMap } from "../../hooks/useMap";
import LogDisplay from "./components/log-display";
import MapControls from "./components/map-controls";
import SimulationControls from "./components/simulation-controls";
import { useTelemetry } from "../../hooks/useTelemetry";
import { useShallow } from "zustand/shallow";
import "../../MapLibre.css";
import { useSimulation } from "../../hooks/useSimulation";
import UavTelemetryDisplay from "./components/uav-telemetry-display";

export default function SimulationPage() {
  const isSimulating = useSimulation((s) => s.isSimulating);
  const simulationFinished = useSimulation((s) => s.simulationFinished);
  const mapContainerRef = useRef<HTMLDivElement>(null);
  const [initMap, updateTrails, updateMarkers, recenter, followTarget] = useMap(
    useShallow((s) => [
      s.init,
      s.updateTrails,
      s.updateMarkers,
      s.recenter,
      s.followTarget,
    ]),
  );
  const [uavs, setonNewRealPoint, subscribe, unsubscribe] = useTelemetry(
    useShallow((s) => [
      s.interpolatedUavs,
      s.setOnNewRealPoint,
      s.subscribe,
      s.unsuscribe,
    ]),
  );

  useEffect(() => {
    if (!mapContainerRef.current) return;
    return initMap(mapContainerRef.current);
  }, [initMap]);
  // }, [initMap, showTrails]);

  useEffect(() => {
    if (!isSimulating) return;
    subscribe();
    return () => {
      unsubscribe();
    };
  }, [subscribe, unsubscribe, isSimulating]);

  useEffect(() => {
    if (!isSimulating) return;
    setonNewRealPoint(updateTrails);
  }, [setonNewRealPoint, updateTrails, isSimulating]);

  useEffect(() => {
    if (!isSimulating) return;
    const uavList = Object.values(uavs);
    updateMarkers(uavList);
    if (!followTarget) return;
    if (uavList.length === 0) return;

    const pos = uavs[0].payload.position;
    recenter(pos.lat, pos.lon);
  }, [uavs, updateMarkers, followTarget, recenter, isSimulating]);

  return (
    <main className="flex" style={{ height: "calc(100vh - 60px)" }}>
      <div ref={mapContainerRef} className="h-full relative flex-1 z-40">
        <MapControls className="absolute top-16 left-2 z-50" />
        <SimulationControls className="absolute top-2 left-2 z-50" />
        <LogDisplay
          className="absolute bottom-2 left-2 right-2"
          onFinishReceived={simulationFinished}
        />
      </div>
      <UavTelemetryDisplay />
    </main>
  );
}
