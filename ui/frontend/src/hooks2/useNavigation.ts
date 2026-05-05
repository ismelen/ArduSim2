import { type ElementType } from "react";
import { create } from "zustand";
import ConfigPage from "../pages2/config/config-page";
import FleetPage from "../pages2/fleet/fleet-page";
import LogsPage from "../pages2/logs-page";
import SimulationPage from "../pages2/simulation/simulation-page";

interface Tab {
  label: string;
  page: ElementType;
}

export const TABS: Tab[] = [
  {
    label: "Config",
    page: ConfigPage,
  },
  {
    label: "Fleet",
    page: FleetPage,
  },
  {
    label: "Simulation",
    page: SimulationPage,
  },
  {
    label: "Logs",
    page: LogsPage,
  },
];

export const useNavigation = create<{
  current: Tab;
  navigateTo(label: Tab): void;
}>((set) => ({
  current: TABS[0],
  navigateTo(tab: Tab) {
    set({ current: tab });
  },
}));
