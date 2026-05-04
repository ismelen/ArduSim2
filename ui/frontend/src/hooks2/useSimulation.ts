import { create } from "zustand";
import hash from "object-hash";
import {
  LoadSimulationConfig,
  SaveSimulationConfig,
  StartSimulation,
} from "../../wailsjs/go/main/App";
import { useFleet, type UAV } from "./useFleet";
import { useConfig, type GeneralConfig } from "./useConfig";
import { domain } from "../../wailsjs/go/models";

interface SimulationState {
  uavs: UAV[];
  generalConfig: GeneralConfig;
  activeMode: string;
}

const DEFAULT_SIMULATION_STATE = {
  uavs: [],
  generalConfig: {},
  activeMode: "LOCAL",
};

interface ValueHash {
  value: SimulationState;
  hash: string;
}

interface State {
  lastConfig: ValueHash;
  undos: ValueHash[];
  redos: ValueHash[];
  lastHash?: string;
  skipNextUpdate: boolean;

  undo(): void;
  redo(): void;
  update(fn: (state: SimulationState) => SimulationState): void;
  loadConfig(): Promise<void>;
  saveConfig(): Promise<void>;
  newConfig(): Promise<void>;
}

export const useSimulation = create<State>((set, get) => ({
  skipNextUpdate: false,
  lastConfig: {
    value: DEFAULT_SIMULATION_STATE,
    hash: "",
  },
  lastHash: "",
  undos: [],
  redos: [],

  undo() {
    const state = get();

    console.log("undos:", state.undos.length);

    const prevConfig = state.undos[state.undos.length - 1];
    const newUndos = state.undos.slice(0, -1);
    set({
      lastConfig: prevConfig,
      undos: newUndos,
      redos: [...state.redos, state.lastConfig],
      skipNextUpdate: true,
    });

    useFleet.getState().loadFleet(prevConfig.value.uavs);
    useConfig
      .getState()
      .loadConfig(prevConfig.value.generalConfig, prevConfig.value.activeMode);
  },

  redo() {
    const state = get();

    const nextConfig = state.redos[state.redos.length - 1];
    const newRedos = state.redos.slice(0, -1);

    set({
      lastConfig: nextConfig,
      undos: [...state.undos, state.lastConfig],
      redos: newRedos,
      skipNextUpdate: true,
    });

    useFleet.getState().loadFleet(nextConfig.value.uavs);
    useConfig
      .getState()
      .loadConfig(nextConfig.value.generalConfig, nextConfig.value.activeMode);
  },

  update(fn: (state: SimulationState) => SimulationState) {
    if (get().skipNextUpdate) {
      set({ skipNextUpdate: false });
      return;
    }
    const currentConfig = get().lastConfig;

    const newState = fn(currentConfig.value);
    const code = hash(newState);
    const newConfig = { value: newState, hash: code };

    set((s) => ({
      lastConfig: newConfig,
      undos: [...s.undos, currentConfig],
      redos: [],
    }));
  },

  async loadConfig() {
    if (get().lastHash && get().lastHash !== get().lastConfig.hash) {
      //TODO: await confirm dialog
    }

    const state = await LoadSimulationConfig();
    const code = hash(state);

    useFleet.getState().loadFleet(state.uavs);
    useConfig.getState().loadConfig(state.generalConfig, state.activeMode);

    set({
      undos: [],
      redos: [],
      lastConfig: {
        value: state,
        hash: code,
      },
      lastHash: code,
    });
  },

  async saveConfig() {
    const state = get().lastConfig;
    await SaveSimulationConfig(
      state.value.uavs.map((e) => domain.UAV.createFrom(e)),
      domain.GeneralConfig.createFrom(state.value.generalConfig),
      state.value.activeMode,
    );
    set({
      lastHash: state.hash,
    });
  },

  async newConfig() {
    //TODO: await confirm dialog

    const state = get();
    if (state.lastConfig.hash !== "") state.saveConfig();

    set({
      undos: [],
      redos: [],
      lastConfig: { value: DEFAULT_SIMULATION_STATE, hash: "" },
      lastHash: "",
      skipNextUpdate: true,
    });
  },

  async startSimulation() {
    const state = get();
    if (state.lastConfig.hash !== "") state.saveConfig();

    const config = state.lastConfig.value;

    await StartSimulation(
      config.uavs.map((e) => domain.UAV.createFrom(e)),
      domain.GeneralConfig.createFrom(config.generalConfig),
      config.activeMode === "LOCAL",
    );
  },
}));
