import Card from "../../../components/card";
import Button from "../../../components/button";
import { cn } from "../../../utils/cn";
import { useMap } from "../../../hooks/useMap";
import { useShallow } from "zustand/shallow";

interface Props {
  className?: string;
}

export default function MapControls({ className }: Props) {
  const [toggleShowTrails, showTrails, toggleFollowTarget, followTarget] =
    useMap(
      useShallow((s) => [
        s.toggleShowTrails,
        s.showTrails,
        s.toggleFollowTarget,
        s.followTarget,
      ]),
    );

  return (
    <Card className={cn("p-1 gap-1 flex", className)}>
      <Button
        onClick={toggleFollowTarget}
        type={followTarget ? "filled" : undefined}
        icon="center_focus_strong"
        className="aspect-square text-base p-2"
      />
      <Button
        type={showTrails ? "filled" : undefined}
        onClick={toggleShowTrails}
        icon="route"
        className="aspect-square text-base p-2"
      />
    </Card>
  );
}
