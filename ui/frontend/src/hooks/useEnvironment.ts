import { create } from 'zustand';

export type ArchMode = 'LOCAL' | 'SWARM';

interface EnvironmentState {
  activeMode: ArchMode;
  masterIP: string;
  showCommand: boolean;
  setActiveMode: (mode: ArchMode) => void;
  setMasterIP: (ip: string) => void;
  setShowCommand: (show: boolean) => void;
}

export const useEnvironment = create<EnvironmentState>((set) => ({
  activeMode: 'LOCAL',
  masterIP: '',
  showCommand: false,
  setActiveMode: (mode) => set({ activeMode: mode }),
  setMasterIP: (ip) => set({ masterIP: ip }),
  setShowCommand: (show) => set({ showCommand: show }),
}));

