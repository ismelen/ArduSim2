import Card from "../../../components/card";
import CardTitle from "../../../components/card-title";
import FormField from "../../../components/form-field";
import TabSelector from "../../../components/tab-selector";
import { useConfig } from "../../../hooks/useConfig";

export default function DeploymentCard() {
  const activeMode = useConfig((s) => s.activeMode);
  const setActiveMode = useConfig((s) => s.setActieMode);
  const netsimInstances = useConfig((s) => s.config.netsimInstances);
  const update = useConfig((s) => s.update);

  return (
    <Card className="flex flex-col gap-2">
      <CardTitle label="Deployment Mode" icon="dns" />
      <TabSelector
        initValue={activeMode}
        onChange={(e) => {
          setActiveMode(e.value);
        }}
        options={[
          { value: "LOCAL", label: "Local", component: <div></div> },
          {
            value: "SWARM",
            label: "Docker Swarm",
            component: <DockerSwarmForm />,
          },
        ]}
      />
      <FormField
        label="Netsim Instances"
        hint="1"
        initValue={String(netsimInstances ?? 1)}
        onChange={(e) =>
          update((s) => ({
            ...s,
            netsimInstances: Math.max(1, parseInt(e) || 1),
          }))
        }
      />
    </Card>
  );
}

function DockerSwarmForm() {
  const { swarmHost } = useConfig((s) => s.config);
  const update = useConfig((s) => s.update);

  return (
    <div className="space-y-2 mt-3">
      <FormField
        label="DOCKER_HOST"
        hint="tcp://ip:port or ssh://user@ip"
        initValue={swarmHost ?? ""}
        onChange={(e) => update((s) => ({ ...s, swarmHost: e }))}
      />
    </div>
  );
}
