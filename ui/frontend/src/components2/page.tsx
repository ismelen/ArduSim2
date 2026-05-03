import type { ReactNode } from "react";
import { cn } from "../utils2/cn";

export default function Page({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={cn("pt-5 px-5", className)}>{children}</div>;
}
