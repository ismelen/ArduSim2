import maplibregl from "maplibre-gl";
import { create } from "zustand";
import type { TelemetryData } from "./useTelemetry";
import { getUavColor } from "../constants/uav-colors";
import { createUAVMarkerElement } from "../utils/uav-marker-builder";

interface State {
  mode3D: boolean;
  showTrails: boolean;
  followTarget: boolean;
  map: maplibregl.Map | null;

  toggleMode3D(): void;
  toggleShowTrails(): void;
  recenter(lat: number, lon: number): void;
  toggleFollowTarget(): void;
  updateTrails(uavId: string, lat: number, lon: number): void;
  updateMarkers(uavs: TelemetryData[]): void;

  init(container: HTMLDivElement): void;
}

export const useMap = create<State>((set, get) => {
  const trails: Record<string, [number, number][]> = {};
  const markers: Record<string, maplibregl.Marker> = {};

  return {
    mode3D: false,
    showTrails: true,
    followTarget: true,
    map: null,

    toggleFollowTarget() {
      set((s) => ({ followTarget: !s.followTarget }));
    },

    updateMarkers(uavs: TelemetryData[]) {
      for (const [idx, uav] of uavs.entries()) {
        const marker = markers[uav.uav_id];
        const pos = uav.payload.position;

        if (!marker) {
          markers[uav.uav_id] = new maplibregl.Marker({
            element: createUAVMarkerElement(pos.heading, pos.alt, idx),
          })
            .setLngLat([pos.lon, pos.lat])
            .addTo(get().map!);

          return;
        }

        const el = marker.getElement();
        const sphere = el.querySelector(".uav-sphere") as HTMLElement;
        if (sphere) sphere.style.transform = `rotate(${pos.heading}deg)`;
        const tag = el.querySelector(".uav-altitude-tag") as HTMLElement;
        if (tag) tag.textContent = `${pos.alt.toFixed(1)}m`;
      }

      const src = get().map?.getSource(
        "uav-trails",
      ) as maplibregl.GeoJSONSource;
      if (!src) return;

      const features = Object.entries(trails).map(([id, coords]) => {
        const uavIdx = uavs.findIndex((s) => s.uav_id === id);
        const uav = uavs[uavIdx];
        const pos = uav?.payload.position;

        if (pos && pos.lon !== 0 && pos.lat !== 0) {
          coords.push([pos.lat, pos.lon]);
        }

        return {
          type: "Feature" as const,
          properties: {
            uavId: id,
            color: getUavColor(uavIdx),
          },
          geometry: {
            type: "LineString" as const,
            coordinates: coords,
          },
        };
      });

      src.setData({ type: "FeatureCollection", features });
    },

    updateTrails(uavId: string, lat: number, lon: number) {
      if (!trails[uavId]) {
        trails[uavId] = [];
      }

      const coords = trails[uavId];
      const last = coords[coords.length - 1];

      if (!last || last[0] !== lat || last[1] !== lon) {
        coords.push([lat, lon]);
      }
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

    toggleMode3D() {
      const { map, mode3D } = get();
      if (mode3D) {
        map?.setMaxPitch(0);
        map?.setMinPitch(0);
        map?.easeTo({ pitch: 0, bearing: 0, duration: 1000 });
      } else {
        map?.setMaxPitch(85);
        map?.setMinPitch(0);
        map?.easeTo({ pitch: 60, bearing: 0, duration: 1000 });
      }
      set({ mode3D: !mode3D });
    },

    init(container: HTMLDivElement) {
      const state = get();

      const map = new maplibregl.Map({
        container: container,
        style: "https://basemaps.cartocdn.com/gl/voyager-gl-style/style.json",
        center: [0, 0],
        zoom: 13,
        pitch: state.mode3D ? 60 : 0,
        maxPitch: 0,
        dragRotate: true,
        scrollZoom: true,
        dragPan: true,
        touchZoomRotate: true,
      });

      map.on("load", () => {
        // Add GeoJSON trace layer
        map.addSource("uav-trails", {
          type: "geojson",
          data: { type: "FeatureCollection", features: [] },
        });
        map.addLayer({
          id: "uav-trails-layer",
          type: "line",
          source: "uav-trails",
          layout: {
            "line-join": "round",
            "line-cap": "round",
            visibility: state.showTrails ? "visible" : "none",
          },
          paint: {
            "line-color": ["get", "color"],
            "line-width": 4,
            "line-opacity": 0.8,
          },
        });

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
      });

      map.addControl(
        new maplibregl.NavigationControl({ showCompass: true }),
        "top-right",
      );

      set({ map: map });

      return () => {
        map.remove();
        set({ map: null });
      };
    },
  };
});
