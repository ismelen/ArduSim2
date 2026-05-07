/* eslint-disable @typescript-eslint/no-explicit-any */
import { create } from "zustand";
import {
  LoadLogEntries,
  LoadLogEntry,
  LoadFile,
} from "../../wailsjs/go/main/App";

interface State {
  logPaths: [string, string][];
  current?: Record<string, any>;
  selectedIdx?: number;
  file?: string;
  loadAll(): Promise<void>;
  loadLogs(idx?: number): Promise<void>;
  loadFile(path: string): Promise<void>;
}

export const useLogs = create<State>((set, get) => ({
  logPaths: [],

  async loadAll() {
    const paths = await LoadLogEntries();
    const logPaths: [string, string][] = [];
    for (const path of paths) {
      const name = path.split(/[/\\]/).pop();
      if (!name) continue;
      logPaths.push([name, path]);
    }

    set({ logPaths: [...logPaths] });
  },

  async loadLogs(idx?: number) {
    if (!idx) {
      set({ current: undefined, selectedIdx: undefined });
      return;
    }
    const path = get().logPaths[idx][1];
    const simulationLogs = await LoadLogEntry(path);
    console.log(simulationLogs);
    set({ current: simulationLogs, selectedIdx: idx });
  },

  async loadFile(path: string) {
    console.log(path);
    set({ file: await LoadFile(path) });
  },
}));
