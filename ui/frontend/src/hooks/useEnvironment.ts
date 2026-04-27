import { create } from 'zustand';

export type ArchMode = 'LOCAL' | 'SWARM';

interface EnvironmentState {
  activeMode: ArchMode;
  masterIP: string;
  masterPort: string;
  setActiveMode: (mode: ArchMode) => void;
  setMasterIP: (ip: string) => void;
  setMasterPort: (port: string) => void;
}

export const useEnvironment = create<EnvironmentState>((set) => ({
  activeMode: 'LOCAL',
  masterIP: '',
  masterPort: '2375',
  setActiveMode: (mode) => set({ activeMode: mode }),
  setMasterIP: (ip) => set({ masterIP: ip }),
  setMasterPort: (port) => set({ masterPort: port }),
}));
