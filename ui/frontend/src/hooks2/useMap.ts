import maplibregl from "maplibre-gl";
import { create } from "zustand";

interface State {
  mode3D: boolean;
  showTrails: boolean;
  followTarget: boolean;
  map: maplibregl.Map | null;

  toggleMode3D(): void;
  toggleShowTrails(): void;
  recenter(lat: number, lon: number): void;

  init(container: HTMLDivElement): void;
}

export const useMap = create<State>((set, get) => ({
  mode3D: false,
  showTrails: true,
  followTarget: true,
  map: null,

  recenter(lat: number, lon: number) {
    get().map?.flyTo({
      center: [lon, lat],
      zoom: 17,
      speed: 1.5
    })
  },

  toggleShowTrails() {
    set((s) => ({ showTrails: !s.showTrails }));
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
}));
