import Card from "../../../components2/card";
import CardTitle from "../../../components2/card-title";
import FilePickerField from "../../../components2/file-picker-field";
import { useConfig } from "../../../hooks2/useConfig";

export default function SpeedProfileCard() {
  const update = useConfig((s) => s.update);
  const { speedProfilePath } = useConfig((s) => s.config);

  return (
    <Card className="flex flex-col gap-2">
      <CardTitle label="Speed Profile" icon="speed" />
      <FilePickerField
        hint="PATH/TO/SPEED_PROFILE.DAT"
        initValue={speedProfilePath}
        onChange={(e) => update((s) => ({ ...s, speedProfilePath: e }))}
      />
    </Card>
  );
}
