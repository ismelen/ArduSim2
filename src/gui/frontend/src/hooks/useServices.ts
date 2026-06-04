import { create } from "zustand";
import { GetAvailableServices } from "../../wailsjs/go/main/App";
import type { domain } from "../../wailsjs/go/models";

interface State {
  services: domain.ServiceType[];
  loadServices(): Promise<void>;
}

export const useServices = create<State>((set, get) => ({
  services: [],
  async loadServices() {
    if (get().services.length !== 0) return;

    const services = await GetAvailableServices();
    set({ services });
  },
}));
