import { SelectSpeedProfile } from "../../../../wailsjs/go/main/App";
import Card from "../../../components/card";
import CardTitle from "../../../components/card-title";
import FilePickerField from "../../../components/file-picker-field";
import { useConfig } from "../../../hooks/useConfig";

export default function SpeedProfileCard() {
  const update = useConfig((s) => s.update);
  const { speedProfilePath } = useConfig((s) => s.config);

  return (
    <Card className="flex flex-col gap-2">
      <CardTitle label="Speed Profile" icon="speed" />
      <FilePickerField
        hint="PATH/TO/SPEED_PROFILE.CSV"
        initValue={speedProfilePath}
        onChange={(e) => update((s) => ({ ...s, speedProfilePath: e }))}
        onBrowse={SelectSpeedProfile}
      />
    </Card>
  );
}
