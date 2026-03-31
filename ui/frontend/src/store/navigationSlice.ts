import type { StateCreator } from 'zustand';

export type TabView = 'ENVIRONMENT' | 'FLEET_CONFIG' | 'ACTIVE_SIM';

export interface NavigationSlice {
  currentTab: TabView;
  isSimulating: boolean;
  setCurrentTab: (tab: TabView) => void;
  startSimulation: () => void;
  exitSimulation: () => void;
}

export const createNavigationSlice: StateCreator<NavigationSlice> = (set) => ({
  currentTab: 'ENVIRONMENT',
  isSimulating: false,
  setCurrentTab: (tab) => set({ currentTab: tab }),
  startSimulation: () => set({ isSimulating: true }),
  exitSimulation: () => set({ isSimulating: false, currentTab: 'FLEET_CONFIG' }),
});
