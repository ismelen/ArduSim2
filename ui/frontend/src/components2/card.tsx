import { type ReactNode } from "react";
import { cn } from "../utils2/cn";

interface Props {
  children?: ReactNode;
  className?: string;
}

export default function Card({ children, className }: Props) {
  return (
    <div
      className={cn(
        "border border-border rounded-md p-3 overflow-clip bg-white shadow-xs",
        className,
      )}
    >
      {children}
    </div>
  );
}
