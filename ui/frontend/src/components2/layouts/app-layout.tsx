import type { ReactNode } from "react";
import Header from "../header";

interface Props {
  children?: ReactNode;
}

export default function AppLayout({ children }: Props) {
  return (
    <div className="bg-background h-screen overflow-y-scroll flex flex-col">
      <Header />
      <div className="flex-1">{children}</div>
    </div>
  );
}
