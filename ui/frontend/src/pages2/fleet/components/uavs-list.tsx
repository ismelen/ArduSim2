import type { domain } from "../../../../wailsjs/go/models";
import Button from "../../../components2/button";
import { useFleet } from "../../../hooks2/useFleet";
import { cn } from "../../../utils2/cn";

export default function UavsList() {
  const uavs = useFleet((s) => s.uavs);
  const addUavs = useFleet((s) => s.addUavs);
  const activeUavIdx = useFleet((s) => s.activeUavIdx);
  const setSelectedIdx = useFleet((s) => s.setSelectedIdx);

  return (
    <aside className="border-r border-border min-w-70 max-w-90 flex-1/4 bg-white">
      <span className="border-b border-border flex flex-row items-center justify-between px-3 py-3 bg-gray">
        <h3 className="font-bold text-xl">Fleet</h3>
        <Button
          icon="add"
          type="filled"
          className="rounded-full"
          onClick={() => addUavs(1)}
        />
      </span>
      <div className="p-2 flex flex-col gap-2">
        {uavs.map((e, idx) => (
          <UavCard
            uav={e}
            selected={idx === activeUavIdx}
            onClick={() => {
              setSelectedIdx(idx);
            }}
          />
        ))}
      </div>
    </aside>
  );
}

interface UavCardProps {
  uav: domain.UAV;
  onClick?(): void;
  selected?: boolean;
}

function UavCard({ uav, onClick, selected }: UavCardProps) {
  return (
    <div
      onClick={() => {
        onClick?.();
      }}
      className={cn(
        `hoverable-gray w-full rounded-md transition-colors duartion-200 flex 
        justify-between items-center px-3 py-3 cursor-pointer
        border border-border shadow-sm`,
        {
          "bg-primary hoverable-primary": selected,
        },
      )}
    >
      <div className="flex-1">
        <p
          className={cn("text-lg font-medium text-dark-gray", {
            "text-onPrimary": selected,
          })}
        >
          Uav {uav.id}
        </p>
        <span className="flex gap-1 items-center">
          {uav.services.map(() => (
            <div
              className={cn("rounded-full aspect-square h-2 bg-dark-gray", {
                "bg-onPrimary": selected,
              })}
            />
          ))}
        </span>
      </div>
      {selected === true ? (
        <span className="material-symbols-rounded text-onPrimary">
          chevron_right
        </span>
      ) : null}
    </div>
  );
}
