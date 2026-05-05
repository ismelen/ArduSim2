import Card from "../../../components/card";
import CardTitle from "../../../components/card-title";
import FormField from "../../../components/form-field";
import TabSelector from "../../../components/tab-selector";
import { useConfig } from "../../../hooks/useConfig";

export default function DeploymentCard() {
  const activeMode = useConfig((s) => s.activeMode);
  const setActiveMode = useConfig((s) => s.setActieMode);

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
    </Card>
  );
}

function DockerSwarmForm() {
  const { swarmHost } = useConfig((s) => s.config);
  const update = useConfig((s) => s.update);

  const parts = (swarmHost ?? "").split(":");
  const ip = parts[0];
  const port = parts[1];

  return (
    <div className="space-y-2 mt-3">
      <FormField
        label="Manager IP Address"
        hint="127.0.0.1"
        initValue={ip}
        onChange={(e) =>
          update((s) => ({ ...s, swarmHost: `${e}:${port ?? ""}` }))
        }
      />
      <FormField
        label="Swarm Port"
        hint="2375"
        initValue={port}
        onChange={(e) =>
          update((s) => ({ ...s, swarmHost: `${ip ?? ""}:${e}` }))
        }
      />
    </div>
  );
}
