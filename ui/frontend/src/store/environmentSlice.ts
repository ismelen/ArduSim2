import type { StateCreator } from 'zustand';

export type ArchMode = 'LOCAL' | 'SWARM';

export interface EnvironmentSlice {
  activeMode: ArchMode;
  masterIP: string;
  setActiveMode: (mode: ArchMode) => void;
  setMasterIP: (ip: string) => void;
}

export const createEnvironmentSlice: StateCreator<EnvironmentSlice> = (set) => ({
  activeMode: 'SWARM',
  masterIP: '192.168.1.100',
  setActiveMode: (mode) => set({ activeMode: mode }),
  setMasterIP: (ip) => set({ masterIP: ip }),
});
