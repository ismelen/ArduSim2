/* eslint-disable react-hooks/set-state-in-effect */
/* eslint-disable @typescript-eslint/no-explicit-any */
import "maplibre-gl/dist/maplibre-gl.css";
import { useEffect, useRef } from "react";
import { useShallow } from "zustand/shallow";
import { EventsOff, EventsOn } from "../../../wailsjs/runtime/runtime";
import "../../MapLibre.css";
import { useMap } from "../../hooks/useMap";
import { useSimulation } from "../../hooks/useSimulation";
import { useTelemetry } from "../../hooks/useTelemetry";
import LogDisplay from "./components/log-display";
import MapControls from "./components/map-controls";
import SimulationControls from "./components/simulation-controls";
import UavTelemetryDisplay from "./components/uav-telemetry-display";

export default function SimulationPage() {
  const isSimulating = useSimulation((s) => s.isSimulating);
  const simulationFinished = useSimulation((s) => s.simulationFinished);
  const mapContainerRef = useRef<HTMLDivElement>(null);
  
  const [initMap, updateTrails, updateMarkers] = useMap(
    useShallow((s) => [
      s.init,
      s.updateTrails,
      s.updateMarkers,
    ]),
  );

  const [interpolatedUavs, setonNewRealPoint, subscribe, unsubscribe, notifyAllReady] =
    useTelemetry(
      useShallow((s) => [
        s.interpolatedUavs,
        s.setOnNewRealPoint,
        s.subscribe,
        s.unsuscribe,
        s.notifyAllReady,
      ]),
    );

  useEffect(() => {
    const READY_EVENT = "simulation:ready";
    const onReady = () => {
      notifyAllReady();
    };
    EventsOn(READY_EVENT, onReady);
    return () => {
      EventsOff(READY_EVENT);
    };
  }, [notifyAllReady]);

  useEffect(() => {
    if (!mapContainerRef.current) return;
    const cleanup = initMap(mapContainerRef.current);
    return cleanup;
  }, [initMap]);

  useEffect(() => {
    if (!isSimulating) return;
    subscribe();
    return () => {
      unsubscribe();
    };
  }, [subscribe, unsubscribe, isSimulating]);

  useEffect(() => {
    setonNewRealPoint(updateTrails);
  }, [setonNewRealPoint, updateTrails]);

  useEffect(() => {
    let id: number;
    const frame = () => {
      const uavList = Object.values(interpolatedUavs());
      updateMarkers(uavList);
      id = requestAnimationFrame(frame);
    };

    id = requestAnimationFrame(frame);
    return () => cancelAnimationFrame(id);
  }, [interpolatedUavs, updateMarkers]);

  return (
    <main className="flex" style={{ height: "calc(100vh - 60px)" }}>
      <div ref={mapContainerRef} className="h-full relative flex-1 z-30">
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
