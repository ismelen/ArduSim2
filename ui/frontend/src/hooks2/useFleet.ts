import { create } from "zustand";
import { domain } from "../../wailsjs/go/models";

interface State {
  uavs: domain.UAV[];
  activeUavIdx: number;
  addUavs(count: number): void;
  deleteUav(): void;
  addService(service: domain.DeployedService): void;
  deleteService(service: domain.DeployedService): void;
  updateService(idx: number, service: domain.DeployedService): void;
  loadFleet(uavs: domain.UAV[]): void;
  setSelectedIdx(idx: number): void;
}

export const useFleet = create<State>((set, get) => ({
  uavs: [domain.UAV.createFrom({ id: 0, services: [] })],
  activeUavIdx: 0,

  updateService(idx: number, service: domain.DeployedService) {
    const uavs = get().uavs;
    const uav = uavs[get().activeUavIdx];
    uav.services[idx] = service;
    uavs[get().activeUavIdx] = uav;

    set({ uavs: [...uavs] });
  },

  setSelectedIdx(idx: number) {
    set({ activeUavIdx: idx });
  },

  loadFleet(uavs: domain.UAV[]) {
    set({ uavs });
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
      uavs.push(
        domain.UAV.createFrom({
          id: i.toString(),
          services: [],
        }),
      );
    }
    set({ uavs: [...uavs] });
  },

  deleteUav() {
    let uavs = get().uavs;
    uavs = [
      ...uavs.slice(0, get().activeUavIdx),
      ...uavs.slice(get().activeUavIdx + 1),
    ];

    if (uavs.length === 0) {
      return set({
        uavs: [domain.UAV.createFrom({ id: 0, services: [] })],
        activeUavIdx: 0,
      });
    }

    if (get().activeUavIdx !== 0) {
      set((s) => ({ activeUavIdx: s.activeUavIdx - 1 }));
    }

    set({ uavs: [...uavs] });
  },

  addService(service: domain.DeployedService) {
    const uavs = get().uavs;
    uavs[get().activeUavIdx].services.push(service);

    set({ uavs });
  },

  deleteService(service: domain.DeployedService) {
    const uavs = get().uavs;
    uavs[get().activeUavIdx].services = uavs[
      get().activeUavIdx
    ].services.filter((e) => e.instanceId !== service.instanceId);

    set({ uavs: [...uavs] });
  },
}));
