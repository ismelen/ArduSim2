import { useState } from "react";
import { cn } from "../../../utils2/cn";
import Button from "../../../components2/button";

interface Props {
  className?: string;
}

export default function LogDisplay({ className }: Props) {
  const [isOpen, setIsOpen] = useState(true);

  return (
    <div
      className={cn(
        `h-1/3 bg-white 
          z-50 rounded-md border-border border overflow-clip`,
        className,
        {
          "h-min": !isOpen,
        },
      )}
    >
      <span
        onClick={() => setIsOpen((s) => !s)}
        className={cn(
          `bg-gray border-b border-border flex items-center 
          justify-between p-1 pl-3 cursor-pointer`,
          { "border-transparent": !isOpen },
        )}
      >
        <p className="text-base font-medium">Log</p>
        <Button icon={isOpen ? "keyboard_arrow_down" : "keyboard_arrow_up"} />
      </span>
      {isOpen && <div className="overflow-y-auto h-full"></div>}
    </div>
  );
}
