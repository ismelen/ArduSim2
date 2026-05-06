/* eslint-disable @typescript-eslint/no-explicit-any */
import "maplibre-gl/dist/maplibre-gl.css";
import { useCallback, useEffect, useState } from "react";
import { useMap } from "../../hooks/useMap";
import { useTelemetry } from "../../hooks/useTelemetry";
import { useShallow } from "zustand/shallow";
import "../../MapLibre.css";
import { useSimulation } from "../../hooks/useSimulation";
import UavTelemetryDisplay from "./components/uav-telemetry-display";
import { ScatterplotLayer, PathLayer } from "@deck.gl/layers";
import { Map } from "react-map-gl/maplibre";
import DeckGL from "@deck.gl/react";
import maplibregl from "maplibre-gl";
import MapControls from "./components/map-controls";
import SimulationControls from "./components/simulation-controls";
import LogDisplay from "./components/log-display";

export default function SimulationPage() {
  const isSimulating = useSimulation((s) => s.isSimulating);
  const simulationFinished = useSimulation((s) => s.simulationFinished);
  const [
    initMap,
    updateTrails,
    updateMarkers,
    recenter,
    followTarget,
    uavTrails,
    uavMarkers,
  ] = useMap(
    useShallow((s) => [
      s.init,
      s.updateTrails,
      s.updateMarkers,
      s.recenter,
      s.followTarget,
      s.uavTrails,
      s.uavMarkers,
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
  const [viewState, setViewState] = useState({
    longitude: -0.349228,
    latitude: 39.481645,
    zoom: 13,
    pitch: 0,
    maxPitch: 85,
    bearing: 0,
    // transitionDuration: 200,
  });

  const init = useCallback((e: any) => initMap(e), [initMap]);

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
    if (!isSimulating) return;
    const uavList = Object.values(uavs);
    updateMarkers(uavList);
    if (!followTarget) return;
    if (uavList.length === 0) return;

    const pos = uavs[0].payload.position;
    recenter(pos.lat, pos.lon);
  }, [uavs, updateMarkers, followTarget, recenter, isSimulating]);

  const onViewStateChange = useCallback(({ viewState }: any) => {
    setViewState(viewState);
  }, []);

  return (
    <main className="flex" style={{ height: "calc(100vh - 60px)" }}>
      <div className="h-full relative flex-1 z-40">
        <DeckGL
          style={{ zIndex: "1" }}
          viewState={viewState}
          onViewStateChange={onViewStateChange}
          controller={true}
          layers={[
            // Capa de trails
            new PathLayer({
              id: "uav-path-layer",
              data: Object.values(uavTrails),
              getPath: (d) => d.path,
              getColor: (d) => d.color,
              widthMinPixels: 2,
              widthUnits: "meters",
              getWidth: 2,
              billboard: true,
            }),

            // Capa de los puntos
            new ScatterplotLayer({
              id: "uav-point-layer",
              data: Object.values(uavMarkers),
              getPosition: (d) => d.position,
              getFillColor: (d) => d.color,
              getRadius: 1,
              radiusMinPixels: 2,
              billboard: true,
            }),
          ]}
        >
          <Map
            mapLib={maplibregl as any}
            mapStyle="https://basemaps.cartocdn.com/gl/voyager-gl-style/style.json"
            onLoad={init}
            style={{ zIndex: "0" }}
          />
          <MapControls className="absolute top-16 left-2 z-50" />
          <SimulationControls className="absolute top-2 left-2 z-50" />
          <LogDisplay
            className="absolute bottom-2 left-2 right-2"
            onFinishReceived={simulationFinished}
          />
        </DeckGL>
      </div>
      <UavTelemetryDisplay />
    </main>
  );
}
