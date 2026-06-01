import { useEffect, useMemo, useState } from "react";
import { GetKmlFirstCoordinate } from "../../../../wailsjs/go/main/App";
import type { domain } from "../../../../wailsjs/go/models";
import CardTitle from "../../../components/card-title";
import FormField from "../../../components/form-field";
import Select from "../../../components/select";
import { useConfig } from "../../../hooks/useConfig";
import { useFleet, type UAV } from "../../../hooks/useFleet";
import { useServices } from "../../../hooks/useServices";
import { cn } from "../../../utils/cn";

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
  const [kmlCoords, setKmlCoords] = useState<domain.Coordinate | undefined>(
    undefined,
  );

  const {
    formationCenterLat,
    formationCenterLon,
    formationSpacing,
    formationCenterMode,
  } = useConfig((s) => s.config);
  const update = useConfig((s) => s.update);
  const uavs = useFleet((s) => s.uavs);
  const services = useServices((s) => s.services);

  const kmlFiles = useMemo<{ filename: string; path: string }[]>(
    () => getKmlFiles(services, uavs),
    [uavs, services],
  );

  useEffect(() => {
    useServices.getState().loadServices();
  }, []);

  // Re-hydrate local kmlCoords so the lat/lon fields are disabled on re-mount
  // when a KML was already selected. Must NOT call update() here — doing so
  // would create a reactive loop (update → formationCenterMode ref changes →
  // effect re-fires) and would race against the user switching back to Custom.
  useEffect(() => {
    if (!formationCenterMode) {
      setKmlCoords(undefined);
      return;
    }

    GetKmlFirstCoordinate(formationCenterMode).then(setKmlCoords);
  }, [formationCenterMode]);

  const handleSelectCoordsSrc = async (value: string | undefined) => {
    if (!value) {
      setKmlCoords(undefined);
      update((s) => ({ ...s, formationCenterMode: value }));
      return;
    }

    const coords = await GetKmlFirstCoordinate(value);
    setKmlCoords(coords);
    update((s) => ({
      ...s,
      formationCenterMode: value,
      formationCenterLat: coords.lat,
      formationCenterLon: coords.lon,
    }));
  };

  return (
    <aside className="border-l border-border min-w-70 max-w-90 flex-1/4 bg-cwhite">
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
        <Select<string>
          key={kmlFiles.map((e) => e.path).join(",")}
          options={[
            { label: "Custom", value: "" },
            ...kmlFiles.map((e) => ({ label: e.filename, value: e.path })),
          ]}
          label="Get coords from"
          initValue={formationCenterMode ?? ""}
          onChange={handleSelectCoordsSrc}
        />
        <span className="flex gap-2">
          <FormField
            initValue={`${kmlCoords?.lat ?? formationCenterLat}`}
            type="number"
            min={0}
            label="Latitude (deg)"
            enabled={kmlCoords === undefined}
            onChange={(e) =>
              update((s) => ({ ...s, formationCenterLat: Number(e) }))
            }
          />
          <FormField
            initValue={`${kmlCoords?.lon ?? formationCenterLon}`}
            type="number"
            min={0}
            label="Longitude (deg)"
            enabled={kmlCoords === undefined}
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
  const { groundFormation } = useConfig((s) => s.config);
  const update = useConfig((s) => s.update);

  return (
    <div className="grid grid-cols-2 grid-rows-2 gap-2">
      {FORMATIONS.map((e) => (
        <div
          key={e.value}
          onClick={() => update((s) => ({ ...s, groundFormation: e.value }))}
          className={cn(
            `border border-border rounded-md p-3 overflow-clip justify-center
              bg-cwhite shadow-xs cursor-pointer hoverable-gray flex flex-col items-center `,
            {
              "bg-primary text-onPrimary hoverable-primary":
                e.value === groundFormation,
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

function getKmlFiles(services: domain.ServiceType[], uavs: UAV[]) {
  const kmlFiles: { filename: string; path: string }[] = [];

  const servicesWithKmlFiles = services.filter((e) =>
    e.schemaRaw.includes('"format"') && e.schemaRaw.includes("kml"),
  );

  for (const uav of uavs) {
    const servicesToSearch = uav.services.filter((e) =>
      servicesWithKmlFiles.find((v) => v.id === e.serviceId),
    );
    for (const serv of servicesToSearch) {
      for (const prop of Object.values(serv.config)) {
        if (typeof prop === "string" && prop.includes(".kml")) {
          kmlFiles.push({
            path: prop as string,
            filename: (prop as string).split(/[/\\]/).pop() ?? "",
          });
        }
      }
    }
  }
  return kmlFiles;
}
