import { type ElementType } from "react";
import { create } from "zustand";
import ConfigPage from "../pages/config/config-page";
import FleetPage from "../pages/fleet/fleet-page";
import LogsPage from "../pages/logs-page";
import SimulationPage from "../pages/simulation/simulation-page";

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
