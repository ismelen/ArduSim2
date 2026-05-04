import { create } from "zustand";
import { useSimulation } from "./useSimulation";

export interface GeneralConfig {
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
  config: GeneralConfig;
  activeMode: string;
  update(fn: (config: GeneralConfig) => GeneralConfig): void;
  loadConfig(config: GeneralConfig, activeMode: string): void;
  setActieMode(value: string): void;
}

export const useConfig = create<State>((set, get) => ({
  config: {},
  activeMode: "LOCAL",

  update(fn: (config: GeneralConfig) => GeneralConfig) {
    const newConfig = fn(get().config);
    set({ config: newConfig });
    useSimulation.getState().update((s) => ({
      ...s,
      generalConfig: newConfig,
    }));
  },

  loadConfig(config: GeneralConfig, activeMode: string) {
    set({ config: config, activeMode: activeMode });
  },

  setActieMode(value: string) {
    set({ activeMode: value });
    useSimulation.getState().update((s) => ({
      ...s,
      activeMode: value,
    }));
  },
}));
