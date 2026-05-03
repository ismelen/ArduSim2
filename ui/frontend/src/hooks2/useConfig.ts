import hash from "object-hash";
import { create } from "zustand";
import { LoadSimulationConfig } from "../../wailsjs/go/main/App";
import { useFleet } from "./useFleet";

interface Config {
  speedProfilePath?: string;
  loggingEnabled?: boolean;
  batteryRestricted?: boolean;
  batteryCapacity?: number;
  verboseLogging?: boolean;
  storeLocalData?: boolean;
  windEnabled?: boolean;
  windDirection?: number;
  windSpeed?: number;
  simulationName?: string;
  originalSimulationName?: string;
  groundFormation?: string;
  formationCenterLat?: number;
  formationCenterLon?: number;
  formationSpacing?: number;
  formationCenterMode?: string;
  swarmHost?: string;
}

interface State {
  config: Config;
  lastHash?: string;
  currentHash?: string;
  activeMode: string;
  update(fn: (config: Config) => Config): void;
  loadConfig(): Promise<void>;
  saveConfig(): Promise<void>;
  setActieMode(value: string): void;
}

export const useConfig = create<State>((set, get) => ({
  config: {},
  activeMode: "LOCAL",

  update(fn: (config: Config) => Config) {
    set((s) => ({ config: fn(s.config) }));
    set((s) => ({ currentHash: hash(s.config) }));
  },

  async loadConfig() {
    if (
      get().lastHash &&
      get().currentHash &&
      get().lastHash !== get().currentHash
    ) {
      //TODO: await confirm dialog
    }

    const state = await LoadSimulationConfig();
    useFleet.getState().loadFleet(state.uavs);
    console.log(state.generalConfig.formationCenterMode);
    set({ config: state.generalConfig, lastHash: hash(state.generalConfig) });
  },

  async saveConfig() {
    set((s) => ({ lastHash: s.currentHash }));
  },

  setActieMode(value: string) {
    set({ activeMode: value });
  },
}));
