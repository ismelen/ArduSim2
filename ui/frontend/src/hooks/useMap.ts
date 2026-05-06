/* eslint-disable @typescript-eslint/no-explicit-any */
import maplibregl from "maplibre-gl";
import { create } from "zustand";
import type { TelemetryData } from "./useTelemetry";
import { getRgbUavColor } from "../constants/uav-colors";

interface State {
  showTerrain: boolean;
  showBuildings: boolean;

  toggleTerrain: () => void;
  toggleBuildings: () => void;

  showTrails: boolean;
  followTarget?: string;
  map: maplibregl.Map | null;
  uavTrails: Record<string, { id: string; path: number[][]; color: number[] }>;
  uavMarkers: Record<
    string,
    { id: string; position: number[]; color: number[] }
  >;
  mode2D: boolean;
  setMode2D(value?: boolean): void;

  toggleShowTrails(): void;
  toggleFollowTarget(value?: string): void;
  updateTrails(uavId: string, lat: number, lon: number, alt: number): void;
  updateMarkers(uavs: TelemetryData[]): void;

  init(event: any): void;
}

export const useMap = create<State>((set, get) => ({
  showTrails: true,
  showBuildings: true,
  showTerrain: false,

  mode2D: false,
  uavTrails: {
    "1": {
      id: "1",
      path: [
        [-0.349228, 39.481645, 0],
        [-0.349228, 39.481645, 10],
        [-0.348228, 39.482645, 15],
        [-0.346228, 39.483645, 15],
      ],
      color: [...getRgbUavColor(1)],
    },
    "2": {
      id: "2",
      path: [
        [-0.347228, 39.483645, 0],
        [-0.347228, 39.483645, 10],
        [-0.345228, 39.484645, 15],
        [-0.343228, 39.485645, 15],
      ],
      color: [...getRgbUavColor(2)],
    },
  },
  uavMarkers: {
    "1": {
      id: "1",
      position: [-0.346228, 39.483645, 15, 120],
      color: [...getRgbUavColor(1)],
    },
    "2": {
      id: "2",
      position: [-0.343228, 39.485645, 15, 90],
      color: [...getRgbUavColor(2)],
    },
  },
  map: null,

  toggleShowTrails: () => set((s) => ({ showTrails: !s.showTrails })),

  toggleTerrain: () => {
    const { map, showTerrain } = get();
    const next = !showTerrain;
    if (map) {
      // Si el próximo estado es ocultar, pasamos null a setTerrain
      map.setTerrain(
        next ? { source: "terrain-source", exaggeration: 1.5 } : null,
      );
    }
    set({ showTerrain: next });
  },

  toggleBuildings: () => {
    const { map, showBuildings } = get();
    const next = !showBuildings;
    if (map && map.getLayer("3d-buildings")) {
      map.setLayoutProperty(
        "3d-buildings",
        "visibility",
        next ? "visible" : "none",
      );
    }
    set({ showBuildings: next });
  },

  setMode2D(value?: boolean) {
    if (value === false) {
      get().map?.setPitch(0);
    }
    set((s) => ({ mode2D: value ?? s.mode2D }));
  },

  toggleFollowTarget(value?: string) {
    set({ followTarget: value });
  },

  updateMarkers(uavs: TelemetryData[]) {
    const markers = get().uavMarkers;

    for (const [idx, uav] of uavs.entries()) {
      const marker = markers[uav.uav_id];
      const pos = uav.payload.position;

      if (!marker) {
        markers[uav.uav_id] = {
          color: getRgbUavColor(idx),
          id: uav.uav_id,
          position: [],
        };
      }

      marker.position = [pos.lat, pos.lon, pos.alt, pos.heading];
    }

    const uavTrails = get().uavTrails;
    Object.entries(uavTrails).map(([id, trail]) => {
      const uavIdx = uavs.findIndex((s) => s.uav_id === id);
      const uav = uavs[uavIdx];
      const pos = uav?.payload.position;

      if (pos && pos.lon !== 0 && pos.lat !== 0 && pos.alt !== 0) {
        trail.path.push([pos.lat, pos.lon, pos.alt]);
      }
    });

    set({ uavTrails: { ...uavTrails }, uavMarkers: { ...markers } });
  },

  updateTrails(uavId: string, lat: number, lon: number, alt: number) {
    const uavTrails = get().uavTrails;

    if (!uavTrails[uavId]) {
      uavTrails[uavId] = {
        id: uavId,
        color: getRgbUavColor(Number(uavId)),
        path: [],
      };
    }

    const { path } = uavTrails[uavId];
    const last = path[path.length - 1];

    if (!last || last[0] !== lat || last[1] !== lon || last[2] != alt) {
      path.push([lat, lon, alt]);
    }

    set({ uavTrails: { ...uavTrails } });
  },

  init(event: any) {
    const map = event.target;

    map.addSource("terrain-source", {
      type: "raster-dem",
      tiles: [
        "https://s3.amazonaws.com/elevation-tiles-prod/terrarium/{z}/{x}/{y}.png",
      ],
      encoding: "terrarium", // ¡Muy importante especificar el encoding para AWS!
      tileSize: 256,
      maxzoom: 14, // Zoom máximo del proveedor de datos
    });
    map.setTerrain(null);

    map.addLayer({
      id: "3d-buildings",
      source: "carto",
      "source-layer": "building",
      type: "fill-extrusion",
      minzoom: 15,
      paint: {
        "fill-extrusion-color": "#d1d5db",
        "fill-extrusion-height": [
          "coalesce",
          ["get", "render_height"],
          ["get", "height"],
          15,
        ],
        "fill-extrusion-base": [
          "coalesce",
          ["get", "render_min_height"],
          ["get", "min_height"],
          0,
        ],
        "fill-extrusion-opacity": 0.5,
        "fill-extrusion-vertical-gradient": true,
      },
    });

    map.setLight({
      anchor: "viewport",
      color: "white",
      intensity: 0.3,
      position: [1.15, 90, 40], // [radial, azimuthal, polar]
    });

    map.setMaxPitch(85);
    map.setMinPitch(0);

    map.addControl(
      new maplibregl.NavigationControl({
        visualizePitch: true,
        visualizeRoll: true,
        showZoom: false,
        showCompass: true,
      }),
    );

    set({ map: map });

    return () => {
      map.remove();
      set({ map: null });
    };
  },
}));
