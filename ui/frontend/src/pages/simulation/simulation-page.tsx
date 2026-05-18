/* eslint-disable react-hooks/set-state-in-effect */
/* eslint-disable @typescript-eslint/no-explicit-any */
import {
  AmbientLight,
  LightingEffect,
  _SunLight as SunLight,
} from "@deck.gl/core";
import { PathLayer } from "@deck.gl/layers";
import { SimpleMeshLayer } from "@deck.gl/mesh-layers";
import DeckGL from "@deck.gl/react";
import { OBJLoader } from "@loaders.gl/obj";
import maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import { useCallback, useEffect, useState } from "react";
import { Map } from "react-map-gl/maplibre";
import { useShallow } from "zustand/shallow";
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
  const [
    initMap,
    updateTrails,
    updateMarkers,
    followTarget,
    uavTrails,
    uavMarkers,
    toggleFollowTarget,
    setMode2D,
    mode2D,
    showTrails,
  ] = useMap(
    useShallow((s) => [
      s.init,
      s.updateTrails,
      s.updateMarkers,
      s.followTarget,
      s.uavTrails,
      s.uavMarkers,
      s.toggleFollowTarget,
      s.setMode2D,
      s.mode2D,
      s.showTrails,
    ]),
  );
  const [interpolatedUavs, setonNewRealPoint, subscribe, unsubscribe] =
    useTelemetry(
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
    maxZoom: 30,
    pitch: 0,
    maxPitch: 85,
    bearing: 0,
  });

  const init = useCallback((e: any) => initMap(e), [initMap]);

  useEffect(() => {
    const frame = () => {
      const uavList = Object.values(interpolatedUavs());
      updateMarkers(uavList);

      requestAnimationFrame(frame);
    };

    const id = requestAnimationFrame(frame);
    return () => cancelAnimationFrame(id);
  }, [interpolatedUavs, updateMarkers]);

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
    if (!followTarget) return;

    const pos = uavMarkers[followTarget]?.position;
    if (!pos) return;

    setViewState((s) => ({
      ...s,
      longitude: pos[0],
      latitude: pos[1],
      zoom: 18,
    }));
  }, [followTarget]);

  useEffect(() => {
    if (!mode2D) return;
    setViewState((s) => ({ ...s, pitch: 0 }));
  }, [mode2D, setViewState]);

  const ambientLight = new AmbientLight({
    color: [255, 255, 255],
    intensity: 1.5, // Bajamos la intensidad
  });
  const sunLight = new SunLight({
    color: [255, 255, 255],
    intensity: 2,
    timestamp: 0,
  });
  const lightingEffect = new LightingEffect({ ambientLight, sunLight });

  const onViewStateChange = useCallback(
    ({ viewState, interactionState }: any) => {
      if (viewState.pitch > 0) setMode2D(false);

      if (followTarget) {
        if (interactionState.isPanning && !interactionState.isZooming) {
          toggleFollowTarget(undefined);
          setViewState(viewState);
          return;
        }
        setViewState((s) => ({
          ...viewState,
          longitude: s.longitude,
          latitude: s.latitude,
        }));
        return;
      }

      setViewState(viewState);
    },
    [toggleFollowTarget, followTarget, setMode2D],
  );

  return (
    <main className="flex" style={{ height: "calc(100vh - 60px)" }}>
      <div className="h-full relative flex-1 z-30">
        <DeckGL
          style={{ zIndex: "1" }}
          viewState={viewState}
          onViewStateChange={onViewStateChange}
          controller={true}
          effects={[lightingEffect]}
          layers={[
            showTrails &&
              new PathLayer({
                id: "uav-path-layer",
                data: Object.values(uavTrails),
                getPath: (d: any) => d.path,
                getColor: (d: any) => d.color,
                widthMinPixels: 2,
                widthUnits: "meters",
                getWidth: 0.25,
                jointRounded: true,
                capRounded: true,
                billboard: true,
                opacity: 0.5,
              }),

            new SimpleMeshLayer({
              id: "uav-mesh-layer",
              data: Object.values(uavMarkers),
              mesh: "/uav1.obj",
              loaders: [OBJLoader],
              getPosition: (d: any) => d.position.slice(0, 3),
              getColor: (d: any) => d.color,
              getScale: [10, 10, 10],
              getOrientation: (d: any) => [0, -d.position[3] || 0, 90],
              _lighting: "phong",
              autoHighlight: true,
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
