import { create } from "zustand";
import { StartSimulation, StopSimulation, LoadSimulationConfig } from "../../wailsjs/go/main/App";
import { useEnvironment } from "./useEnvironment";
import { useFleet } from "./useFleet";
import { useNavigation } from "./useNavigation";

export interface GeneralConfigState {
  speedProfilePath: string;
  loggingEnabled: boolean;
  batteryRestricted: boolean;
  batteryCapacity: number;
  verboseLogging: boolean;
  storeLocalData: boolean;
  windEnabled: boolean;
  windDirection: number;
  windSpeed: number;
}

interface ConfigState extends GeneralConfigState {
  setSpeedProfilePath: (path: string) => void;
  setLoggingEnabled: (enabled: boolean) => void;
  setBatteryRestricted: (restricted: boolean) => void;
  setBatteryCapacity: (capacity: number) => void;
  setVerboseLogging: (enabled: boolean) => void;
  setStoreLocalData: (enabled: boolean) => void;
  setWindEnabled: (enabled: boolean) => void;
  setWindDirection: (direction: number) => void;
  setWindSpeed: (speed: number) => void;
  handleStartSimulation: () => Promise<void>;
  handleExitSimulation: () => Promise<void>;
  handleLoadSimulation: () => Promise<void>;
}

export const useConfig = create<ConfigState>((set, get) => ({
  speedProfilePath: "",
  loggingEnabled: false,
  batteryRestricted: false,
  batteryCapacity: 4500,
  verboseLogging: false,
  storeLocalData: true,
  windEnabled: false,
  windDirection: 0,
  windSpeed: 0,

  setSpeedProfilePath: (speedProfilePath) => set({ speedProfilePath }),
  setLoggingEnabled: (loggingEnabled) => set({ loggingEnabled }),
  setBatteryRestricted: (batteryRestricted) => set({ batteryRestricted }),
  setBatteryCapacity: (batteryCapacity) => set({ batteryCapacity }),
  setVerboseLogging: (verboseLogging) => set({ verboseLogging }),
  setStoreLocalData: (storeLocalData) => set({ storeLocalData }),
  setWindEnabled: (windEnabled) => set({ windEnabled }),
  setWindDirection: (windDirection) => set({ windDirection }),
  setWindSpeed: (windSpeed) => set({ windSpeed }),

  handleStartSimulation: async () => {
    const { uavs } = useFleet.getState();
    const { activeMode } = useEnvironment.getState();
    const { startSimulation, exitSimulation } = useNavigation.getState();
    const { handleStartSimulation, ...config } = get();

    // Navigate immediately to the simulation view
    startSimulation();

    try {
      await StartSimulation(uavs as any, config, activeMode, activeMode === "LOCAL");
    } catch (err) {
      console.error("Failed to start simulation:", err);
      // Show error message
      alert(`SIMULATION_ERROR: ${err instanceof Error ? err.message : String(err)}`);
      // Navigate back to configuration
      exitSimulation();
      throw err;
    }
  },


  handleExitSimulation: async () => {
    const { exitSimulation } = useNavigation.getState();
    try {
      await StopSimulation();
      exitSimulation();
    } catch (err) {
      console.error("Failed to stop simulation:", err);
      alert(`STOP_ERROR: ${err instanceof Error ? err.message : String(err)}`);
      throw err;
    }
  },

  handleLoadSimulation: async () => {
    try {
      const state = await LoadSimulationConfig();
      if (!state) return; // User cancelled

      const { loadFleet } = useFleet.getState();
      const { setActiveMode } = useEnvironment.getState();
      const { 
        setSpeedProfilePath, setLoggingEnabled, setBatteryRestricted, 
        setBatteryCapacity, setVerboseLogging, setStoreLocalData, 
        setWindEnabled, setWindDirection, setWindSpeed 
      } = get();

      // Load Fleet
      loadFleet(state.uavs);

      // Load Environment
      setActiveMode(state.activeMode as any);

      // Load General Config
      setSpeedProfilePath(state.generalConfig.speedProfilePath);
      setLoggingEnabled(state.generalConfig.loggingEnabled);
      setBatteryRestricted(state.generalConfig.batteryRestricted);
      setBatteryCapacity(state.generalConfig.batteryCapacity);
      setVerboseLogging(state.generalConfig.verboseLogging);
      setStoreLocalData(state.generalConfig.storeLocalData);
      setWindEnabled(state.generalConfig.windEnabled);
      setWindDirection(state.generalConfig.windDirection);
      setWindSpeed(state.generalConfig.windSpeed);

    } catch (err) {
      console.error("Failed to load simulation:", err);
      alert(`LOAD_ERROR: ${err instanceof Error ? err.message : String(err)}`);
      throw err;
    }

  },
}));
