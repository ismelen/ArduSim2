import { create } from "zustand";
import { GetAvailableServices } from "../../wailsjs/go/main/App";

interface ServicesState {
  availableServices: any[];
  loading: boolean;
  fetchServices: () => Promise<void>;
}

export const useServices = create<ServicesState>((set) => ({
  availableServices: [],
  loading: false,
  fetchServices: async () => {
    set({ loading: true });
    try {
      const res = await GetAvailableServices();
      if (res) set({ availableServices: res });
    } finally {
      set({ loading: false });
    }
  },
}));

export function buildDefaultValuesFromSchema(
  schemaRaw: string,
): Record<string, any> {
  try {
    const schema = JSON.parse(schemaRaw);
    if (!schema.properties) return {};
    return Object.fromEntries(
      Object.entries<any>(schema.properties).map(([key, prop]) => [
        key,
        prop.default !== undefined ? prop.default : "",
      ]),
    );
  } catch {
    return {};
  }
}
