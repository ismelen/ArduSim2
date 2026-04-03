import { create } from 'zustand';
import { useServices, buildDefaultValuesFromSchema } from './useServices';

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

interface FleetState {
  uavs: UAV[];
  activeUavId: string;
  setActiveUavId: (id: string) => void;
  addUavs: (count: number) => void;
  deleteUav: (id: string) => void;
  deployService: (uavId: string, service: Omit<DeployedService, 'instanceId'>, editingId: string | null) => void;
  deleteService: (uavId: string, instanceId: string) => void;
  deployToAll: (service: Omit<DeployedService, 'instanceId'>) => void;
  syncAll: () => void;
  clearAll: () => void;
  loadFleet: (uavs: UAV[]) => void;

  // Interactive Configuration State (Moved from useFleetConfig hook)
  selectedServiceId: string;
  formValues: Record<string, any>;
  editingInstanceId: string | null;
  showDeployBox: boolean;

  setSelectedServiceId: (id: string) => void;
  setFormValues: (values: Record<string, any> | ((prev: any) => any)) => void;
  setShowDeployBox: (show: boolean) => void;
  resetDeployForm: () => void;
  startEditing: (svc: any) => void;
  handleDeploy: () => void;
  handleDeployToAll: () => void;
}

const makeId = () => Math.random().toString(36).substring(7);

export const useFleet = create<FleetState>((set, get) => ({
  uavs: [
    { id: '1', services: [] },
    { id: '2', services: [] },
    { id: '3', services: [] },
  ],
  activeUavId: '1',

  selectedServiceId: '',
  formValues: {},
  editingInstanceId: null,
  showDeployBox: false,

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
    const activeUavId = state.activeUavId === id ? (uavs[0]?.id ?? '') : state.activeUavId;
    return { uavs, activeUavId };
  }),

  deployService: (uavId, service, editingId) => set((state) => ({
    uavs: state.uavs.map(u => {
      if (u.id !== uavId) return u;
      const services = [...u.services];
      if (editingId) {
        const idx = services.findIndex(s => s.instanceId === editingId);
        if (idx !== -1) services[idx] = { instanceId: editingId, ...service };
      } else {
        services.push({ instanceId: makeId(), ...service });
      }
      return { ...u, services };
    }),
  })),

  deleteService: (uavId, instanceId) => set((state) => ({
    uavs: state.uavs.map(u =>
      u.id !== uavId ? u : { ...u, services: u.services.filter(s => s.instanceId !== instanceId) },
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

  loadFleet: (uavs) => set({ uavs, activeUavId: uavs[0]?.id ?? '' }),

  // Configuration Actions
  setSelectedServiceId: (id) => {
    const { editingInstanceId } = get();
    set({ selectedServiceId: id });
    if (!editingInstanceId) {
      const { availableServices } = useServices.getState();
      const svc = availableServices.find(s => s.id === id);
      if (svc?.schemaRaw) set({ formValues: buildDefaultValuesFromSchema(svc.schemaRaw) });
    }
  },

  setFormValues: (values) => {
    if (typeof values === 'function') {
      set((state) => ({ formValues: values(state.formValues) }));
    } else {
      set({ formValues: values });
    }
  },

  setShowDeployBox: (show) => set({ showDeployBox: show, editingInstanceId: null }),

  resetDeployForm: () => {
    const { availableServices } = useServices.getState();
    const { selectedServiceId } = get();
    set({ editingInstanceId: null, showDeployBox: false });
    const svc = availableServices.find(s => s.id === selectedServiceId);
    if (svc?.schemaRaw) set({ formValues: buildDefaultValuesFromSchema(svc.schemaRaw) });
  },

  startEditing: (svc) => {
    set({
      showDeployBox: false,
      selectedServiceId: svc.serviceId,
      formValues: svc.config,
      editingInstanceId: svc.instanceId,
    });
  },

  handleDeploy: () => {
    const { selectedServiceId, formValues, editingInstanceId, activeUavId, deployService, resetDeployForm } = get();
    const { availableServices } = useServices.getState();
    const selectedSvc = availableServices.find(s => s.id === selectedServiceId);
    if (!selectedServiceId) return;

    deployService(activeUavId, {
      serviceId: selectedServiceId,
      serviceTitle: selectedSvc?.title ?? selectedServiceId,
      config: { ...formValues },
    }, editingInstanceId);
    resetDeployForm();
  },

  handleDeployToAll: () => {
    const { selectedServiceId, formValues, deployToAll, resetDeployForm } = get();
    const { availableServices } = useServices.getState();
    const selectedSvc = availableServices.find(s => s.id === selectedServiceId);
    if (!selectedServiceId) return;

    deployToAll({
      serviceId: selectedServiceId,
      serviceTitle: selectedSvc?.title ?? selectedServiceId,
      config: { ...formValues }
    });
    resetDeployForm();
  }
}));

export const uavDisplayName = (id: string): string => `UAV-${id.padStart(2, '0')}`;
