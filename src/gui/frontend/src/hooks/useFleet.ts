/* eslint-disable @typescript-eslint/no-explicit-any */
import { create } from "zustand";
import { useSimulationConfig } from "./useSimulationConfig";

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
  mixer?: DeployedService | null;
  controller?: DeployedService | null;
  speed?: number | null;
  batteryCapacity?: number | null;
  homeOverride?: { lat: number; lon: number } | null;
  arduPilotInstance?: string | null;
}

interface State {
  uavs: UAV[];
  activeUavIdx: number;
  addUavs(count: number): void;
  deleteUav(): void;
  addService(service: DeployedService): void;
  deleteService(service: DeployedService): void;
  updateService(idx: number, service: DeployedService): void;
  updateUav(idx: number, patch: Partial<UAV>): void;
  loadFleet(uavs: UAV[]): void;
  setSelectedIdx(idx: number): void;
  cloneUav(): void;
}

export const useFleet = create<State>((set, get) => ({
  uavs: [{ id: "1", services: [] }],
  activeUavIdx: 0,

  updateService(idx: number, service: DeployedService) {
    const uavs = get().uavs;
    const activeIdx = get().activeUavIdx;
    
    const updatedUavs = uavs.map((uav, uavIdx) => {
      if (uavIdx !== activeIdx) return uav;
      const updatedServices = uav.services.map((s, sIdx) => 
        sIdx === idx ? service : s
      );
      return { ...uav, services: updatedServices };
    });

    set({ uavs: updatedUavs });
    useSimulationConfig.getState().update((s) => ({ ...s, uavs: updatedUavs }));
  },

  updateUav(idx: number, patch: Partial<UAV>) {
    const uavs = get().uavs;
    const updatedUavs = uavs.map((uav, uavIdx) => {
      if (uavIdx !== idx) return uav;
      return { ...uav, ...patch };
    });

    set({ uavs: updatedUavs });
    useSimulationConfig.getState().update((s) => ({ ...s, uavs: updatedUavs }));
  },

  setSelectedIdx(idx: number) {
    set({ activeUavIdx: idx });
  },

  loadFleet(uavs: UAV[]) {
    if (uavs.length === 0) {
      uavs = [{ id: "0", services: [] }];
    }

    set({ uavs: uavs });
    useSimulationConfig.getState().update((s) => ({ ...s, uavs: uavs }));
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
    useSimulationConfig.getState().update((s) => ({ ...s, uavs: uavs }));
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
    useSimulationConfig.getState().update((s) => ({ ...s, uavs: uavs }));
  },

  addService(service: DeployedService) {
    const uavs = get().uavs;
    const activeIdx = get().activeUavIdx;
    
    const updatedUavs = uavs.map((uav, idx) => 
      idx === activeIdx 
        ? { ...uav, services: [...uav.services, service] }
        : uav
    );

    set({ uavs: updatedUavs });
    useSimulationConfig.getState().update((s) => ({ ...s, uavs: updatedUavs }));
  },

  deleteService(service: DeployedService) {
    const uavs = get().uavs;
    uavs[get().activeUavIdx].services = uavs[
      get().activeUavIdx
    ].services.filter((e) => e.instanceId !== service.instanceId);

    set({ uavs: [...uavs] });
    useSimulationConfig.getState().update((s) => ({ ...s, uavs: uavs }));
  },

  cloneUav() {
    const state = get();
    const uavs = [...state.uavs];
    const activeUav = uavs[state.activeUavIdx];

    if (!activeUav) return;

    let lastId: number;
    if (uavs.length === 0) {
      lastId = 0;
    } else {
      lastId = Number(uavs[uavs.length - 1].id);
    }

    const clonedServices = activeUav.services.map((s) => ({
      ...s,
      instanceId: crypto.randomUUID(),
    }));

    uavs.push({
      id: (lastId + 1).toString(),
      services: clonedServices,
    });

    set({ uavs });
    useSimulationConfig.getState().update((s) => ({ ...s, uavs }));
  },
}));
