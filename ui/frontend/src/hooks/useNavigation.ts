import { create } from "zustand";

export type TabView =
  | "ENVIRONMENT"
  | "FLEET_CONFIG"
  | "GENERAL_CONFIG"
  | "ACTIVE_SIM";

interface NavigationState {
  currentTab: TabView;
  previousTab: TabView;
  isSimulating: boolean;
  setCurrentTab: (tab: TabView) => void;
  startSimulation: () => void;
  exitSimulation: () => void;
}

export const useNavigation = create<NavigationState>((set, get) => ({
  currentTab: "ENVIRONMENT",
  previousTab: "ENVIRONMENT",
  isSimulating: false,
  setCurrentTab: (tab) => set({ currentTab: tab }),
  startSimulation: () =>
    set({ isSimulating: true, previousTab: get().currentTab }),
  exitSimulation: () =>
    set({ isSimulating: false, currentTab: get().previousTab }),
}));
