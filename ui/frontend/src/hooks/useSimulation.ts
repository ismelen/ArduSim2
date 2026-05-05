import { create } from "zustand";
import hash from "object-hash";
import {
  DiscardCurrentRun,
  LoadSimulationConfig,
  SaveSimulationConfig,
  SendAlgorithmCommand,
  StartSimulation,
  StopSimulation,
} from "../../wailsjs/go/main/App";
import { useFleet, type UAV } from "./useFleet";
import { useConfig, type GeneralConfig } from "./useConfig";
import { domain } from "../../wailsjs/go/models";
import { useDialog } from "./useDialog";

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
  isSimulating: boolean;

  undo(): void;
  redo(): void;
  update(fn: (state: SimulationState) => SimulationState): void;
  loadConfig(): Promise<void>;
  saveConfig(): Promise<void>;
  newConfig(): Promise<void>;

  start(targets: string[]): void;
  pause(targets: string[]): void;
  stop(targets: string[]): void;
  exit(): Promise<boolean>;
}

export const useSimulation = create<State>((set, get) => ({
  isSimulating: false,
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
    if (!get().lastHash || get().lastHash !== get().lastConfig.hash) {
      const action = await useDialog.getState().show({
        title: "Unsaved changes",
        text: "There are unsaved changes. Loading a new configuration will result in the loss of these modifications",
        buttons: [
          { label: "Cancel" },
          { label: "Accept", type: "outlined" },
          { label: "Save & Accept", type: "filled" },
        ],
      });
      switch (action) {
        case 0:
          return;
        case 2:
          await get().saveConfig();
      }
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
    if (!get().lastHash || get().lastHash !== get().lastConfig.hash) {
      const action = await useDialog.getState().show({
        title: "Unsaved changes",
        text: "There are unsaved changes. Loading a new configuration will result in the loss of these modifications",
        buttons: [
          { label: "Cancel" },
          { label: "Accept", type: "outlined" },
          { label: "Save & Accept", type: "filled" },
        ],
      });
      switch (action) {
        case 0:
          return;
        case 2:
          await get().saveConfig();
      }
    }

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
    if (get().isSimulating) {
      const action = await useDialog.getState().show({
        title: "Simulation in progress",
        text: "There is a simulation in progress. Starting a new simulation wil destroy current.",
        buttons: [{ label: "Cancel" }, { label: "Accept", type: "filled" }],
      });
      switch (action) {
        case 0:
          return;
        case 1: {
          const exitSucces = await get().exit();
          if (!exitSucces) return;
        }
      }
    }

    const state = get();
    if (state.lastConfig.hash !== "") state.saveConfig();

    const config = state.lastConfig.value;

    await StartSimulation(
      config.uavs.map((e) => domain.UAV.createFrom(e)),
      domain.GeneralConfig.createFrom(config.generalConfig),
      config.activeMode === "LOCAL",
    );
    set({ isSimulating: true });
  },

  async start(targets: string[]) {
    for (const target of targets) {
      handleSendAlgorithmCommand(target, "start");
    }
  },
  async pause(targets: string[]) {
    for (const target of targets) {
      handleSendAlgorithmCommand(target, "pause");
    }
  },
  async stop(targets: string[]) {
    for (const target of targets) {
      handleSendAlgorithmCommand(target, "stop");
    }
  },
  async exit() {
    let keepLogs = true;
    const action = await useDialog.getState().show({
      title: "Are you sure?",
      text: "The current simulation will be closed. Do you want to save logs?",
      buttons: [
        { label: "Cancel" },
        { label: "Don't save", type: "outlined" },
        { label: "Save logs", type: "filled" },
      ],
    });
    switch (action) {
      case 0:
        return false;
      case 1:
        keepLogs = false;
    }
    await StopSimulation();
    if (!keepLogs) {
      await DiscardCurrentRun(
        domain.GeneralConfig.createFrom(get().lastConfig.value.generalConfig),
      );
    }

    set({ isSimulating: false });
    return true;
  },
}));

async function handleSendAlgorithmCommand(serviceId: string, command: string) {
  try {
    await SendAlgorithmCommand(serviceId, command);
  } catch (err) {
    console.error("Failed to send algorithm command:", err);
    alert(`COMMAND_ERROR: ${err instanceof Error ? err.message : String(err)}`);
    throw err;
  }
}
