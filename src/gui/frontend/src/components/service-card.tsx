import type { domain } from "../../wailsjs/go/models";
import Button from "./button";

export default function ServiceCard({
  service,
  onSelect,
  onDelete,
}: {
  service: domain.DeployedService;
  onSelect?(): void;
  onDelete?(): void;
}) {
  return (
    <div
      className="border border-border rounded-md hoverable-gray shadow-sm
      flex justify-between items-center bg-cwhite px-3 py-1.5 cursor-pointer"
    >
      <p className="text-lg">{service.serviceTitle}</p>
      <span className="flex gap-2">
        <Button icon="edit" onClick={onSelect} />
        {onDelete && <Button icon="close" onClick={onDelete} />}
      </span>
    </div>
  );
}
