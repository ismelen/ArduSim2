/* eslint-disable @typescript-eslint/no-explicit-any */
import maplibregl from "maplibre-gl";
import { create } from "zustand";
import type { TelemetryData } from "./useTelemetry";
import { getRgbUavColor } from "../constants/uav-colors";

interface State {
  showTrails: boolean;
  followTarget: boolean;
  map: maplibregl.Map | null;
  uavTrails: Record<string, { id: string; path: number[][]; color: number[] }>;
  uavMarkers: Record<
    string,
    { id: string; position: number[]; color: number[] }
  >;

  toggleShowTrails(): void;
  recenter(lat: number, lon: number): void;
  toggleFollowTarget(): void;
  updateTrails(uavId: string, lat: number, lon: number, alt: number): void;
  updateMarkers(uavs: TelemetryData[]): void;

  init(event: any): void;
}

export const useMap = create<State>((set, get) => ({
  uavTrails: {
    "1": {
      id: "1",
      path: [
        [-0.349228, 39.481645, 0],
        [-0.349228, 39.481645, 10],
        [-0.348228, 39.482645, 15],
        [-0.346228, 39.483645, 15],
      ],
      color: [16, 185, 129],
    },
  },
  uavMarkers: {
    "1": {
      id: "1",
      position: [-0.346228, 39.483645, 15],
      color: [249, 115, 22],
    },
  },
  showTrails: true,
  followTarget: true,
  map: null,

  toggleFollowTarget() {
    set((s) => ({ followTarget: !s.followTarget }));
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

      marker.position = [pos.lat, pos.lon, pos.alt];

      //TODO: heading and altitude
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

    set({ uavTrails: { ...uavTrails } });
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

  recenter(lat: number, lon: number) {
    get().map?.flyTo({
      center: [lon, lat],
      zoom: 17,
      speed: 1.5,
    });
  },

  toggleShowTrails() {
    const { map, showTrails } = get();
    set((s) => ({ showTrails: !s.showTrails }));

    map?.setLayoutProperty(
      "uav-trails-layer",
      "visibility",
      showTrails ? "visible" : "none",
    );
  },

  init(event: any) {
    const map = event.target;

    console.log("asdf");
    map.addSource("terrain-source", {
      type: "raster-dem",
      tiles: [
        "https://s3.amazonaws.com/elevation-tiles-prod/terrarium/{z}/{x}/{y}.png",
      ],
      encoding: "terrarium", // ¡Muy importante especificar el encoding para AWS!
      tileSize: 256,
      maxzoom: 14, // Zoom máximo del proveedor de datos
    });
    map.setTerrain({ source: "terrain-source", exaggeration: 1.5 });

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

    set({ map: map });

    return () => {
      map.remove();
      set({ map: null });
    };
  },
}));
