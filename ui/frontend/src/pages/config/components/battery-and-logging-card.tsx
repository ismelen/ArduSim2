import Card from "../../../components/card";
import CardTitle from "../../../components/card-title";
import Checkbox from "../../../components/checkbox";
import FormField from "../../../components/form-field";
import { useConfig } from "../../../hooks/useConfig";

export default function BatteryAndLoggingCard() {
  const { batteryCapacity, loggingEnabled, batteryRestricted } = useConfig(
    (s) => s.config,
  );
  const update = useConfig((s) => s.update);

  return (
    <Card className="flex flex-col gap-2">
      <CardTitle label="Battery & Logging" icon="checklist_rtl" />
      <span className="flex flex-row items-center justify-between border border-border rounded-md py-1.5 px-3">
        <div>
          <label className="text-base ">Arducopter logging</label>
          <p className="text-dark-gray text-sm">
            Enable detailed telemetry dumps
          </p>
        </div>
        <Checkbox
          onChange={(e) => update((s) => ({ ...s, loggingEnabled: e }))}
          value={loggingEnabled}
        />
      </span>
      <FormField
        label="Battery Restriction"
        hint="5000"
        type="number"
        initValue={batteryCapacity?.toString()}
        onChange={(e) => update((s) => ({ ...s, batteryCapacity: Number(e) }))}
        enabled={batteryRestricted ?? false}
        prefix={
          <Checkbox
            onChange={(e) => update((s) => ({ ...s, batteryRestricted: e }))}
            value={batteryRestricted}
          />
        }
        suffix={<p>mAh</p>}
      />
    </Card>
  );
}
