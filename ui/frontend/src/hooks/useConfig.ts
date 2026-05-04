import { create } from "zustand";
import {
  StartSimulation,
  StopSimulation,
  LoadSimulationConfig,
  SendAlgorithmCommand,
  SaveSimulationConfig,
  DiscardCurrentRun,
} from "../../wailsjs/go/main/App";
import { useEnvironment } from "./useEnvironment";
import { useFleet } from "./useFleet";
import { useNavigation } from "./useNavigation";

export interface GeneralConfigState {
  simulationName: string;
  originalSimulationName: string;
  speedProfilePath: string;
  loggingEnabled: boolean;
  batteryRestricted: boolean;
  batteryCapacity: number;
  verboseLogging: boolean;
  storeLocalData: boolean;
  windEnabled: boolean;
  windDirection: number;
  windSpeed: number;
  groundFormation: string;
  formationCenterLat: number;
  formationCenterLon: number;
  formationSpacing: number;
  formationCenterMode: string;
  /** Remote Docker API endpoint for Swarm deployments (e.g. "192.168.1.10:2375") */
  swarmHost: string;
}

interface ConfigState extends GeneralConfigState {
  setSimulationName: (name: string) => void;
  setOriginalSimulationName: (name: string) => void;
  setSpeedProfilePath: (path: string) => void;
  setLoggingEnabled: (enabled: boolean) => void;
  setBatteryRestricted: (restricted: boolean) => void;
  setBatteryCapacity: (capacity: number) => void;
  setVerboseLogging: (enabled: boolean) => void;
  setStoreLocalData: (enabled: boolean) => void;
  setWindEnabled: (enabled: boolean) => void;
  setWindDirection: (direction: number) => void;
  setWindSpeed: (speed: number) => void;
  setGroundFormation: (formation: string) => void;
  setFormationCenterLat: (lat: number) => void;
  setFormationCenterLon: (lon: number) => void;
  setFormationSpacing: (spacing: number) => void;
  setFormationCenterMode: (mode: string) => void;
  isExiting: boolean;
  handleStartSimulation: () => Promise<void>;
  handleExitSimulation: () => Promise<void>;
  handleLoadSimulation: () => Promise<void>;
  handleSaveSimulation: () => Promise<void>;
  handleSendAlgorithmCommand: (
    serviceId: string,
    command: string,
  ) => Promise<void>;
}

export const useConfig = create<ConfigState>((set, get) => ({
  simulationName: "",
  originalSimulationName: "",
  speedProfilePath: "",
  loggingEnabled: false,
  batteryRestricted: false,
  batteryCapacity: 4500,
  verboseLogging: false,
  storeLocalData: true,
  windEnabled: false,
  windDirection: 0,
  windSpeed: 0,
  groundFormation: "LINEAR",
  formationCenterLat: 39.482594,
  formationCenterLon: -0.346265,
  formationSpacing: 5.0,
  formationCenterMode: "CUSTOM",
  swarmHost: "",
  isExiting: false,

  setSimulationName: (simulationName) => set({ simulationName }),
  setOriginalSimulationName: (originalSimulationName) =>
    set({ originalSimulationName }),
  setSpeedProfilePath: (speedProfilePath) => set({ speedProfilePath }),
  setLoggingEnabled: (loggingEnabled) => set({ loggingEnabled }),
  setBatteryRestricted: (batteryRestricted) => set({ batteryRestricted }),
  setBatteryCapacity: (batteryCapacity) => set({ batteryCapacity }),
  setVerboseLogging: (verboseLogging) => set({ verboseLogging }),
  setStoreLocalData: (storeLocalData) => set({ storeLocalData }),
  setWindEnabled: (windEnabled) => set({ windEnabled }),
  setWindDirection: (windDirection) => set({ windDirection }),
  setWindSpeed: (windSpeed) => set({ windSpeed }),
  setGroundFormation: (groundFormation) => set({ groundFormation }),
  setFormationCenterLat: (formationCenterLat) => set({ formationCenterLat }),
  setFormationCenterLon: (formationCenterLon) => set({ formationCenterLon }),
  setFormationSpacing: (formationSpacing) => set({ formationSpacing }),
  setFormationCenterMode: (formationCenterMode) => set({ formationCenterMode }),

  handleStartSimulation: async () => {
    const { uavs } = useFleet.getState();
    const { activeMode, masterIP, masterPort } = useEnvironment.getState();
    const { startSimulation, exitSimulation } = useNavigation.getState();
    const { handleStartSimulation, ...config } = get();

    // Compose the swarm host address and persist it in the config sent to the backend.
    const isLocal = activeMode === "LOCAL";
    const swarmHost = isLocal ? "" : `${masterIP}:${masterPort}`;
    const configWithSwarm = { ...config, swarmHost };

    // Navigate immediately to the simulation view
    startSimulation();

    try {
      await StartSimulation(uavs as any, configWithSwarm, isLocal);
    } catch (err) {
      console.error("Failed to start simulation:", err);
      alert(
        `SIMULATION_ERROR: ${err instanceof Error ? err.message : String(err)}`,
      );
      exitSimulation();
      throw err;
    }
  },

  handleExitSimulation: async () => {
    const { exitSimulation } = useNavigation.getState();
    const keepLogs = confirm(
      "¿Deseas guardar los logs y telemetría de esta simulación?",
    );

    set({ isExiting: true });

    try {
      await StopSimulation();

      if (!keepLogs) {
        const {
          handleSaveSimulation,
          handleStartSimulation,
          handleLoadSimulation,
          handleExitSimulation,
          handleSendAlgorithmCommand,
          ...config
        } = get();
        await DiscardCurrentRun(config);
      }

      exitSimulation();
    } catch (err) {
      console.error("Failed to stop simulation:", err);
      alert(`STOP_ERROR: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      set({ isExiting: false });
    }
  },

  handleLoadSimulation: async () => {
    try {
      const state = await LoadSimulationConfig();
      if (!state) return; // User cancelled

      const { loadFleet } = useFleet.getState();
      const { setActiveMode } = useEnvironment.getState();
      const {
        setSimulationName,
        setOriginalSimulationName,
        setSpeedProfilePath,
        setLoggingEnabled,
        setBatteryRestricted,
        setBatteryCapacity,
        setVerboseLogging,
        setStoreLocalData,
        setWindEnabled,
        setWindDirection,
        setWindSpeed,
        setGroundFormation,
        setFormationCenterLat,
        setFormationCenterLon,
        setFormationSpacing,
        setFormationCenterMode,
      } = get();

      // Load Fleet
      loadFleet(state.uavs);

      // Load Environment (including swarm host if present)
      setActiveMode(state.activeMode as any);
      if (state.generalConfig.swarmHost) {
        const [ip, port] = state.generalConfig.swarmHost.split(":");
        useEnvironment.getState().setMasterIP(ip ?? "");
        useEnvironment.getState().setMasterPort(port ?? "2375");
      }

      // Load General Config
      setSimulationName(
        state.generalConfig.simulationName ||
          state.generalConfig.originalSimulationName ||
          "",
      );
      setOriginalSimulationName(
        state.generalConfig.originalSimulationName ||
          state.generalConfig.simulationName ||
          "",
      );
      setSpeedProfilePath(state.generalConfig.speedProfilePath);
      setLoggingEnabled(state.generalConfig.loggingEnabled);
      setBatteryRestricted(state.generalConfig.batteryRestricted);
      setBatteryCapacity(state.generalConfig.batteryCapacity);
      setVerboseLogging(state.generalConfig.verboseLogging);
      setStoreLocalData(state.generalConfig.storeLocalData);
      setWindEnabled(state.generalConfig.windEnabled);
      setWindDirection(state.generalConfig.windDirection);
      setWindSpeed(state.generalConfig.windSpeed);
      setGroundFormation(state.generalConfig.groundFormation || "LINEAR");
      setFormationCenterLat(
        state.generalConfig.formationCenterLat || 39.482594,
      );
      setFormationCenterLon(
        state.generalConfig.formationCenterLon || -0.346265,
      );
      setFormationSpacing(state.generalConfig.formationSpacing || 5.0);
      setFormationCenterMode(
        state.generalConfig.formationCenterMode || "CUSTOM",
      );
    } catch (err) {
      console.error("Failed to load simulation:", err);
      alert(`LOAD_ERROR: ${err instanceof Error ? err.message : String(err)}`);
      throw err;
    }
  },

  handleSaveSimulation: async () => {
    const { uavs } = useFleet.getState();
    const { activeMode } = useEnvironment.getState();
    const {
      handleSaveSimulation,
      handleStartSimulation,
      handleLoadSimulation,
      handleExitSimulation,
      handleSendAlgorithmCommand,
      ...config
    } = get();
    try {
      await SaveSimulationConfig(uavs as any, config, activeMode);
    } catch (err) {
      console.error("Failed to save simulation:", err);
      alert(`SAVE_ERROR: ${err instanceof Error ? err.message : String(err)}`);
      throw err;
    }
  },

  handleSendAlgorithmCommand: async (serviceId: string, command: string) => {
    try {
      await SendAlgorithmCommand(serviceId, command);
    } catch (err) {
      console.error("Failed to send algorithm command:", err);
      alert(
        `COMMAND_ERROR: ${err instanceof Error ? err.message : String(err)}`,
      );
      throw err;
    }
  },
}));
