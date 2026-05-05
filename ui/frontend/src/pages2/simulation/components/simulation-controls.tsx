import { useMemo } from "react";
import Card from "../../../components2/card";
import SplitButton from "../../../components2/split-button";
import { useFleet, type UAV } from "../../../hooks2/useFleet";
import Button from "../../../components2/button";
import type { SelectableValue } from "../../../components2/select";
import { cn } from "../../../utils2/cn";
import { useSimulation } from "../../../hooks2/useSimulation";
import { useShallow } from "zustand/shallow";

interface Props {
  className?: string;
}

export default function SimulationControls({ className }: Props) {
  const uavs = useFleet((s) => s.uavs);
  const availableServices = useMemo(() => getAvailabeServices(uavs), [uavs]);
  const [start, pause, stop, exit] = useSimulation(
    useShallow((s) => [s.start, s.pause, s.stop, s.exit]),
  );

  return (
    <Card className={cn("p-1 flex gap-1 w-min overflow-visible", className)}>
      <SplitButton
        icon="play_arrow"
        label="Start"
        options={availableServices}
        onClick={() => start(availableServices.map((e) => e.value))}
        onSelectOption={(e) => start([e])}
      />
      <SplitButton
        icon="pause"
        label="Pause"
        color="#4d4949"
        options={availableServices}
        onClick={() => pause(availableServices.map((e) => e.value))}
        onSelectOption={(e) => pause([e])}
      />
      <SplitButton
        icon="stop"
        label="Stop"
        color="#a83e3e"
        options={availableServices}
        onClick={() => stop(availableServices.map((e) => e.value))}
        onSelectOption={(e) => stop([e])}
      />
      <Button icon="exit_to_app" label="Exit" onClick={exit} />
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
