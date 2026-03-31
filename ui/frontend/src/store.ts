import { create } from 'zustand';
import { createEnvironmentSlice, type EnvironmentSlice } from './store/environmentSlice';
import { createFleetSlice, type FleetSlice } from './store/fleetSlice';
import { createNavigationSlice, type NavigationSlice } from './store/navigationSlice';

export type RootStore = NavigationSlice & EnvironmentSlice & FleetSlice;

export const useAppStore = create<RootStore>()((...a) => ({
  ...createNavigationSlice(...a),
  ...createEnvironmentSlice(...a),
  ...createFleetSlice(...a),
}));

// Re-exporting types for convenience
export type { ArchMode } from './store/environmentSlice';
export type { DeployedService, UAV } from './store/fleetSlice';
export type { TabView } from './store/navigationSlice';

/** Derives the display name of a UAV from its numeric ID, e.g. "UAV-03". */
export const uavDisplayName = (id: string): string =>
  `UAV-${id.padStart(2, '0')}`;
