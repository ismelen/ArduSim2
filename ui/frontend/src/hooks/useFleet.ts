/* eslint-disable @typescript-eslint/no-explicit-any */
import { create } from "zustand";
import { useSimulation } from "./useSimulation";

export interface DeployedService {
  instanceId: string;
  serviceId: string;
  folderName: string;
  serviceTitle: string;
  config: Record<string, any>;
}

export interface UAV {
  id: string;
  services: DeployedService[];
}

interface State {
  uavs: UAV[];
  activeUavIdx: number;
  addUavs(count: number): void;
  deleteUav(): void;
  addService(service: DeployedService): void;
  deleteService(service: DeployedService): void;
  updateService(idx: number, service: DeployedService): void;
  loadFleet(uavs: UAV[]): void;
  setSelectedIdx(idx: number): void;
}

export const useFleet = create<State>((set, get) => ({
  uavs: [
    { id: "1", services: [] },
    { id: "2", services: [] },
    { id: "3", services: [] },
    { id: "4", services: [] },
    { id: "5", services: [] },
    { id: "6", services: [] },
  ],
  activeUavIdx: 0,

  updateService(idx: number, service: DeployedService) {
    const uavs = get().uavs;
    const uav = uavs[get().activeUavIdx];
    uav.services[idx] = service;
    uavs[get().activeUavIdx] = uav;

    set({ uavs: [...uavs] });
    useSimulation.getState().update((s) => ({ ...s, uavs: uavs }));
  },

  setSelectedIdx(idx: number) {
    set({ activeUavIdx: idx });
  },

  loadFleet(uavs: UAV[]) {
    if (uavs.length === 0) {
      uavs = [{ id: "0", services: [] }];
    }

    set({ uavs: uavs });
    useSimulation.getState().update((s) => ({ ...s, uavs: uavs }));
  },

  addUavs(count: number) {
    const uavs = get().uavs;
    let lastId: number;
    if (uavs.length === 0) {
      lastId = 0;
    } else {
      lastId = Number(uavs[uavs.length - 1].id);
    }
    for (let i = lastId + 1; i <= lastId + count; i++) {
      uavs.push({
        id: i.toString(),
        services: [],
      });
    }
    set({ uavs: [...uavs] });
    useSimulation.getState().update((s) => ({ ...s, uavs: uavs }));
  },

  deleteUav() {
    let uavs = get().uavs;
    uavs = [
      ...uavs.slice(0, get().activeUavIdx),
      ...uavs.slice(get().activeUavIdx + 1),
    ];

    if (uavs.length === 0) {
      return set({
        uavs: [{ id: "0", services: [] }],
        activeUavIdx: 0,
      });
    }

    if (get().activeUavIdx !== 0) {
      set((s) => ({ activeUavIdx: s.activeUavIdx - 1 }));
    }

    set({ uavs: [...uavs] });
    useSimulation.getState().update((s) => ({ ...s, uavs: uavs }));
  },

  addService(service: DeployedService) {
    const uavs = get().uavs;
    uavs[get().activeUavIdx].services.push(service);

    set({ uavs });
    useSimulation.getState().update((s) => ({ ...s, uavs: uavs }));
  },

  deleteService(service: DeployedService) {
    const uavs = get().uavs;
    uavs[get().activeUavIdx].services = uavs[
      get().activeUavIdx
    ].services.filter((e) => e.instanceId !== service.instanceId);

    set({ uavs: [...uavs] });
    useSimulation.getState().update((s) => ({ ...s, uavs: uavs }));
  },
}));
