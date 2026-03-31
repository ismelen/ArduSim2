import type { StateCreator } from 'zustand';

export interface DeployedService {
  instanceId:   string;
  serviceId:    string;
  serviceTitle: string;
  config:       Record<string, any>;
}

export interface UAV {
  id:       string;
  services: DeployedService[];
}

export interface FleetSlice {
  uavs:           UAV[];
  activeUavId:    string;
  setActiveUavId: (id: string) => void;
  addUavs:        (count: number) => void;
  deleteUav:      (id: string) => void;
  deployService:  (
    uavId:             string,
    service:           Omit<DeployedService, 'instanceId'>,
    editingInstanceId: string | null,
  ) => void;
  deleteService:  (uavId: string, instanceId: string) => void;
  deployToAll:    (service: Omit<DeployedService, 'instanceId'>) => void;
  syncAll:        () => void;
  clearAll:       () => void;
}

const makeId = () => Math.random().toString(36).substring(7);

export const createFleetSlice: StateCreator<FleetSlice> = (set, get) => ({
  uavs: [
    { id: '1', services: [] },
    { id: '2', services: [] },
    { id: '3', services: [] },
  ],
  activeUavId: '1',
  setActiveUavId: (id) => set({ activeUavId: id }),

  addUavs: (count) => set((state) => {
    const next = [...state.uavs];
    for (let i = 0; i < count; i++) {
      const maxId = Math.max(0, ...next.map(u => parseInt(u.id) || 0));
      next.push({ id: (maxId + 1).toString(), services: [] });
    }
    return { uavs: next };
  }),

  deleteUav: (id) => set((state) => {
    const uavs = state.uavs.filter(u => u.id !== id);
    const activeUavId = state.activeUavId === id
      ? (uavs[0]?.id ?? '')
      : state.activeUavId;
    return { uavs, activeUavId };
  }),

  deployService: (uavId, service, editingInstanceId) => set((state) => ({
    uavs: state.uavs.map(u => {
      if (u.id !== uavId) return u;
      const services = [...u.services];
      if (editingInstanceId) {
        const idx = services.findIndex(s => s.instanceId === editingInstanceId);
        if (idx !== -1) services[idx] = { instanceId: editingInstanceId, ...service };
      } else {
        services.push({ instanceId: makeId(), ...service });
      }
      return { ...u, services };
    }),
  })),

  deleteService: (uavId, instanceId) => set((state) => ({
    uavs: state.uavs.map(u =>
      u.id !== uavId
        ? u
        : { ...u, services: u.services.filter(s => s.instanceId !== instanceId) },
    ),
  })),

  deployToAll: (service) => set((state) => ({
    uavs: state.uavs.map(u => ({
      ...u,
      services: [...u.services, { instanceId: makeId(), ...service }],
    })),
  })),

  syncAll: () => {
    const { uavs, activeUavId } = get();
    const source = uavs.find(u => u.id === activeUavId)?.services ?? [];
    if (source.length === 0) return;
    set({
      uavs: uavs.map(u => ({
        ...u,
        services: source.map(s => ({ ...s, instanceId: makeId() })),
      })),
    });
  },

  clearAll: () => set((state) => ({
    uavs: state.uavs.map(u => ({ ...u, services: [] })),
  })),
});
