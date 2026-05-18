/* eslint-disable @typescript-eslint/no-explicit-any */
import maplibregl from "maplibre-gl";
import { create } from "zustand";
import { getRgbUavColor } from "../constants/uav-colors";
import type { TelemetryData } from "./useTelemetry";

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
  reset(): void;
}

export const useMap = create<State>((set, get) => ({
  showTrails: true,
  showBuildings: true,
  showTerrain: false,

  mode2D: false,
  uavTrails: {},
  uavMarkers: {},
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
      let marker = markers[uav.uav_id!];
      const pos = uav.position;

      if (!marker) {
        marker = {
          color: getRgbUavColor(idx),
          id: uav.uav_id!,
          position: [],
        };
        markers[uav.uav_id!] = marker;
      }

      marker.position = [pos.lon, pos.lat, pos.alt, pos.heading];
    }

    // const uavTrails = get().uavTrails;
    // Object.entries(uavTrails).map(([id, trail]) => {
    //   const uavIdx = uavs.findIndex((s) => s.uav_id === id);
    //   const uav = uavs[uavIdx];
    //   const pos = uav?.payload.position;

    //   if (pos && pos.lon !== 0 && pos.lat !== 0 && pos.alt !== 0) {
    //     // TODO: Check direction
    //     trail.path.push([pos.lon, pos.lat, pos.alt]);
    //   }
    // });

    // set({ uavTrails: { ...uavTrails }, uavMarkers: { ...markers } });
    set({ uavMarkers: { ...markers } });
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
    if (!last || last[0] !== lon || last[1] !== lat || last[2] != alt) {
      //TODO: Check direction
      path.push([lon, lat, alt]);
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

    if (map.getSource("carto")) {
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
    } else {
      console.warn("La fuente 'carto' no existe en el mapa base cargado.");
    }

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

  reset() {
    set({ uavMarkers: {}, uavTrails: {} });
  },
}));
