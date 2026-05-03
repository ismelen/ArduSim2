import CardTitle from "../../../components2/card-title";
import FormField from "../../../components2/form-field";
import { useConfig } from "../../../hooks2/useConfig";
import { cn } from "../../../utils2/cn";

const FORMATIONS = [
  {
    value: "LINEAR",
    icon: "linear_scale",
  },
  {
    value: "MATRIX",
    icon: "grid_view",
  },
  {
    value: "CIRCLE",
    icon: "circle",
  },
  {
    value: "RANDOM",
    icon: "scatter_plot",
  },
];

export default function SwarmFormation() {
  const { formationCenterLat, formationCenterLon, formationSpacing } =
    useConfig((s) => s.config);
  const update = useConfig((s) => s.update);

  return (
    <aside className="border-l border-border min-w-70 max-w-90 flex-1/4 bg-white">
      <span className="border-b border-border flex flex-row items-center justify-between px-3 py-3 bg-gray">
        <CardTitle label="Swarm Formation" icon="grid_3x3" />
      </span>
      <div className="p-2 flex flex-col gap-2">
        <FormationModeSelection />
        <FormField
          initValue={`${formationSpacing}`}
          type="number"
          min={0}
          label="Formation Spacing"
          suffix={<p>m</p>}
          onChange={(e) =>
            update((s) => ({ ...s, formationSpacing: Number(e) }))
          }
        />
        <span className="flex gap-2">
          <FormField
            initValue={`${formationCenterLat}`}
            type="number"
            min={0}
            label="Latitude (deg)"
            onChange={(e) =>
              update((s) => ({ ...s, formationCenterLat: Number(e) }))
            }
          />
          <FormField
            initValue={`${formationCenterLon}`}
            type="number"
            min={0}
            label="Longitude (deg)"
            onChange={(e) =>
              update((s) => ({ ...s, formationCenterLon: Number(e) }))
            }
          />
        </span>
      </div>
    </aside>
  );
}

function FormationModeSelection() {
  const { formationCenterMode } = useConfig((s) => s.config);
  const update = useConfig((s) => s.update);

  return (
    <div className="grid grid-cols-2 grid-rows-2 gap-2">
      {FORMATIONS.map((e) => (
        <div
          onClick={() =>
            update((s) => ({ ...s, formationCenterMode: e.value }))
          }
          className={cn(
            `border border-border rounded-md p-3 overflow-clip justify-center
              bg-white shadow-xs cursor-pointer hoverable-gray flex flex-col items-center `,
            {
              "bg-primary text-onPrimary hoverable-primary":
                e.value === formationCenterMode,
            },
          )}
        >
          <span className="material-symbols-rounded">{e.icon}</span>
          <p>{e.value}</p>
        </div>
      ))}
    </div>
  );
}
