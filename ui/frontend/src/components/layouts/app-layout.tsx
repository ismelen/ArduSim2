import type { ReactNode } from "react";
import { useTheme } from "../../hooks/useTheme";
import { cn } from "../../utils/cn";
import Header from "../header";

interface Props {
  children?: ReactNode;
}

export default function AppLayout({ children }: Props) {
  const isNight = useTheme((s) => s.isNight);

  return (
    <div
      className={cn(
        "bg-background h-screen overflow-y-auto flex flex-col text-cblack font-firasans",
        {
          dark: isNight,
        },
      )}
    >
      <Header />
      <div className="flex-1">{children}</div>
    </div>
  );
}
