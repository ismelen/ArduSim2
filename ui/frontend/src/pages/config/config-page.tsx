import Button from "../../components/button";
import Page from "../../components/page";
import PageTitle from "../../components/page-title";
import { useSimulationPersistence } from "../../hooks/useSimulationPersistence";
import BatteryAndLoggingCard from "./components/battery-and-logging-card";
import DeploymentCard from "./components/deployment-card";
import SimNameCard from "./components/sim-name-card";
import SpeedProfileCard from "./components/speed-profile-card";
import WindCard from "./components/wind-card";

export default function ConfigPage() {
  const loadConfig = useSimulationPersistence((s) => s.loadConfig);

  return (
    <Page>
      <span className="flex flex-row items-end justify-between">
        <PageTitle
          title="General Config"
          subtitle="Define environment parameters and deployment infrastructure."
        />
        <Button
          label="Load Simulation Config"
          icon="folder_open"
          type="outlined"
          onClick={loadConfig}
        />
      </span>
      <div className="flex flex-row gap-3 mt-5">
        <div className="flex-2/3 flex flex-col gap-3">
          <SimNameCard />
          <WindCard />
          <SpeedProfileCard />
        </div>
        <div className="flex-1/3 flex flex-col gap-3">
          <DeploymentCard />
          <BatteryAndLoggingCard />
        </div>
      </div>
    </Page>
  );
}
