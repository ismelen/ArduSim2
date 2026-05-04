import { useMemo } from "react";
import Card from "../../../components2/card";
import SplitButton from "../../../components2/split-button";
import { useFleet, type UAV } from "../../../hooks2/useFleet";
import Button from "../../../components2/button";
import type { SelectableValue } from "../../../components2/select";
import { cn } from "../../../utils2/cn";

interface Props {
  className?: string;
}

export default function SimulationControls({ className }: Props) {
  const uavs = useFleet((s) => s.uavs);
  const availableServices = useMemo(() => getAvailabeServices(uavs), [uavs]);

  return (
    <Card className={cn("p-1 flex gap-1 w-min overflow-visible", className)}>
      <SplitButton
        icon="play_arrow"
        label="Start"
        options={availableServices}
      />
      <SplitButton
        icon="pause"
        label="Pause"
        color="#4d4949"
        options={availableServices}
      />
      <SplitButton
        icon="stop"
        label="Stop"
        color="#bd2323"
        options={availableServices}
      />
      <Button icon="exit_to_app" label="Exit" />
    </Card>
  );
}

function getAvailabeServices(uavs: UAV[]): SelectableValue<string>[] {
  const services: Record<string, string> = {};

  for (const uav of uavs) {
    for (const serv of uav.services) {
      services[serv.instanceId] = serv.serviceTitle;
    }
  }

  return Object.entries(services).map(([k, v]) => ({
    value: k,
    label: v,
  }));
}
