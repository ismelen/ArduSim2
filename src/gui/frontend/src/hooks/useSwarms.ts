import { create } from "zustand";
import { useSimulationConfig } from "./useSimulationConfig";
import { domain } from "../../wailsjs/go/models";

export type Swarm = domain.Swarm;
export type UAV = domain.UAV;
export type DeployedService = domain.DeployedService;

interface State {
  swarms: Swarm[];
  activeSwarmIdx: number;
  activeUavIdx: number | null;

  addSwarms(count: number): void;
  deleteSwarm(): void;
  updateSwarm(idx: number, patch: Partial<Swarm>): void;

  addUavs(count: number): void;
  deleteUav(): void;
  cloneUav(): void;
  updateUav(idx: number, patch: Partial<UAV>): void;

  addService(service: DeployedService): void;
  deleteService(service: DeployedService): void;
  updateService(idx: number, service: DeployedService): void;

  loadSwarms(swarms: Swarm[]): void;
  setSelectedSwarm(idx: number): void;
  setSelectedUav(idx: number | null): void;
}

export const useSwarms = create<State>((set, get) => ({
  swarms: [{
    id: "1",
    uavs: [{ id: "1", services: [] as DeployedService[] }] as UAV[],
    groundFormation: "grid",
    formationCenterLat: 0,
    formationCenterLon: 0,
    formationSpacing: 5,
    formationCenterMode: "AUTO"
  } as Swarm],
  activeSwarmIdx: 0,
  activeUavIdx: null,

  updateSwarm(idx: number, patch: Partial<Swarm>) {
    const swarms = get().swarms;
    const updatedSwarms = swarms.map((swarm, swarmIdx) => {
      if (swarmIdx !== idx) return swarm;
      return { ...swarm, ...patch };
    });
    set({ swarms: updatedSwarms as Swarm[] });
    useSimulationConfig.getState().update((s) => ({ ...s, swarms: updatedSwarms as Swarm[] }));
  },

  addSwarms(count: number) {
    const swarms = get().swarms;
    let lastId = 0;
    if (swarms.length > 0) {
      lastId = Number(swarms[swarms.length - 1].id);
      if (isNaN(lastId)) lastId = swarms.length;
    }
    const newSwarms = [...swarms];
    for (let i = lastId + 1; i <= lastId + count; i++) {
      newSwarms.push({
        id: i.toString(),
        uavs: [{ id: "1", services: [] as DeployedService[] }] as UAV[],
        groundFormation: "grid",
        formationCenterLat: 0,
        formationCenterLon: 0,
        formationSpacing: 5,
        formationCenterMode: "AUTO"
      } as Swarm);
    }
    set({ swarms: newSwarms });
    useSimulationConfig.getState().update((s) => ({ ...s, swarms: newSwarms }));
  },

  deleteSwarm() {
    let swarms = get().swarms;
    swarms = [
      ...swarms.slice(0, get().activeSwarmIdx),
      ...swarms.slice(get().activeSwarmIdx + 1),
    ];

    if (swarms.length === 0) {
      const defaultSwarm = {
        id: "1",
        uavs: [{ id: "1", services: [] as DeployedService[] }] as UAV[],
        groundFormation: "grid",
        formationCenterLat: 0,
        formationCenterLon: 0,
        formationSpacing: 5,
        formationCenterMode: "AUTO"
      } as Swarm;
      set({
        swarms: [defaultSwarm],
        activeSwarmIdx: 0,
        activeUavIdx: null
      });
      useSimulationConfig.getState().update((s) => ({ ...s, swarms: [defaultSwarm] }));
      return;
    }

    let activeSwarmIdx = get().activeSwarmIdx;
    if (activeSwarmIdx >= swarms.length) {
      activeSwarmIdx = swarms.length - 1;
    }
    set({ swarms, activeSwarmIdx, activeUavIdx: null });
    useSimulationConfig.getState().update((s) => ({ ...s, swarms }));
  },

  loadSwarms(swarms: Swarm[]) {
    if (!swarms || swarms.length === 0) {
      swarms = [{
        id: "1",
        uavs: [{ id: "1", services: [] as DeployedService[] }] as UAV[],
        groundFormation: "grid",
        formationCenterLat: 0,
        formationCenterLon: 0,
        formationSpacing: 5,
        formationCenterMode: "AUTO"
      } as Swarm];
    }
    set({ swarms, activeSwarmIdx: 0, activeUavIdx: null });
    useSimulationConfig.getState().update((s) => ({ ...s, swarms }));
  },

  setSelectedSwarm(idx: number) {
    set({ activeSwarmIdx: idx, activeUavIdx: null });
  },

  setSelectedUav(idx: number | null) {
    set({ activeUavIdx: idx });
  },

  updateService(idx: number, service: DeployedService) {
    const activeSwarmIdx = get().activeSwarmIdx;
    const activeUavIdx = get().activeUavIdx;
    if (activeUavIdx === null) return;
    
    const swarms = get().swarms;
    const updatedSwarms = swarms.map((swarm, sIdx) => {
      if (sIdx !== activeSwarmIdx) return swarm;
      const updatedUavs = swarm.uavs.map((uav, uIdx) => {
        if (uIdx !== activeUavIdx) return uav;
        const updatedServices = uav.services.map((s, svcIdx) => 
          svcIdx === idx ? service : s
        );
        return { ...uav, services: updatedServices } as UAV;
      });
      return { ...swarm, uavs: updatedUavs } as Swarm;
    });

    set({ swarms: updatedSwarms });
    useSimulationConfig.getState().update((s) => ({ ...s, swarms: updatedSwarms }));
  },

  updateUav(idx: number, patch: Partial<UAV>) {
    const activeSwarmIdx = get().activeSwarmIdx;
    const swarms = get().swarms;
    
    const updatedSwarms = swarms.map((swarm, sIdx) => {
      if (sIdx !== activeSwarmIdx) return swarm;
      const updatedUavs = swarm.uavs.map((uav, uIdx) => {
        if (uIdx !== idx) return uav;
        return { ...uav, ...patch } as UAV;
      });
      return { ...swarm, uavs: updatedUavs } as Swarm;
    });

    set({ swarms: updatedSwarms });
    useSimulationConfig.getState().update((s) => ({ ...s, swarms: updatedSwarms }));
  },

  addUavs(count: number) {
    const swarms = get().swarms;
    const activeSwarmIdx = get().activeSwarmIdx;
    const swarm = swarms[activeSwarmIdx];
    
    let lastId = 0;
    if (swarm.uavs.length > 0) {
      lastId = Number(swarm.uavs[swarm.uavs.length - 1].id);
      if (isNaN(lastId)) lastId = swarm.uavs.length;
    }
    
    const newUavs = [...swarm.uavs];
    for (let i = lastId + 1; i <= lastId + count; i++) {
      newUavs.push({
        id: i.toString(),
        services: [] as DeployedService[],
      } as UAV);
    }
    
    const updatedSwarms = swarms.map((s, idx) => 
      idx === activeSwarmIdx ? { ...s, uavs: newUavs } as Swarm : s
    );

    set({ swarms: updatedSwarms });
    useSimulationConfig.getState().update((s) => ({ ...s, swarms: updatedSwarms }));
  },

  deleteUav() {
    const activeSwarmIdx = get().activeSwarmIdx;
    const activeUavIdx = get().activeUavIdx;
    if (activeUavIdx === null) return;
    
    const swarms = get().swarms;
    let uavs = swarms[activeSwarmIdx].uavs;
    
    uavs = [
      ...uavs.slice(0, activeUavIdx),
      ...uavs.slice(activeUavIdx + 1),
    ];

    if (uavs.length === 0) {
      uavs = [{ id: "1", services: [] as DeployedService[] } as UAV];
    }

    const updatedSwarms = swarms.map((s, idx) => 
      idx === activeSwarmIdx ? { ...s, uavs } as Swarm : s
    );

    let newUavIdx: number | null = activeUavIdx;
    if (newUavIdx >= uavs.length) {
      newUavIdx = uavs.length - 1;
    }

    set({ swarms: updatedSwarms, activeUavIdx: newUavIdx });
    useSimulationConfig.getState().update((s) => ({ ...s, swarms: updatedSwarms }));
  },

  addService(service: DeployedService) {
    const activeSwarmIdx = get().activeSwarmIdx;
    const activeUavIdx = get().activeUavIdx;
    if (activeUavIdx === null) return;
    
    const swarms = get().swarms;
    const updatedSwarms = swarms.map((swarm, sIdx) => {
      if (sIdx !== activeSwarmIdx) return swarm;
      const updatedUavs = swarm.uavs.map((uav, uIdx) => {
        if (uIdx !== activeUavIdx) return uav;
        return { ...uav, services: [...uav.services, service] } as UAV;
      });
      return { ...swarm, uavs: updatedUavs } as Swarm;
    });

    set({ swarms: updatedSwarms });
    useSimulationConfig.getState().update((s) => ({ ...s, swarms: updatedSwarms }));
  },

  deleteService(service: DeployedService) {
    const activeSwarmIdx = get().activeSwarmIdx;
    const activeUavIdx = get().activeUavIdx;
    if (activeUavIdx === null) return;
    
    const swarms = get().swarms;
    const updatedSwarms = swarms.map((swarm, sIdx) => {
      if (sIdx !== activeSwarmIdx) return swarm;
      const updatedUavs = swarm.uavs.map((uav, uIdx) => {
        if (uIdx !== activeUavIdx) return uav;
        return { 
          ...uav, 
          services: uav.services.filter((e) => e.instanceId !== service.instanceId)
        } as UAV;
      });
      return { ...swarm, uavs: updatedUavs } as Swarm;
    });

    set({ swarms: updatedSwarms });
    useSimulationConfig.getState().update((s) => ({ ...s, swarms: updatedSwarms }));
  },

  cloneUav() {
    const activeSwarmIdx = get().activeSwarmIdx;
    const activeUavIdx = get().activeUavIdx;
    if (activeUavIdx === null) return;
    
    const swarms = get().swarms;
    const swarm = swarms[activeSwarmIdx];
    const uavs = [...swarm.uavs];
    const activeUav = uavs[activeUavIdx];

    if (!activeUav) return;

    let lastId = 0;
    if (uavs.length > 0) {
      lastId = Number(uavs[uavs.length - 1].id);
      if (isNaN(lastId)) lastId = uavs.length;
    }

    const clonedServices = activeUav.services.map((s) => ({
      ...s,
      instanceId: crypto.randomUUID(),
    } as DeployedService));

    uavs.push({
      id: (lastId + 1).toString(),
      services: clonedServices,
      mixer: activeUav.mixer,
      speed: activeUav.speed,
      batteryCapacity: activeUav.batteryCapacity,
      homeOverride: activeUav.homeOverride,
      arduPilotInstance: activeUav.arduPilotInstance,
    } as UAV);

    const updatedSwarms = swarms.map((s, idx) => 
      idx === activeSwarmIdx ? { ...s, uavs } as Swarm : s
    );

    set({ swarms: updatedSwarms });
    useSimulationConfig.getState().update((s) => ({ ...s, swarms: updatedSwarms }));
  },
}));
