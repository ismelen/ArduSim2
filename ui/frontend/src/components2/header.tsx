import { TABS, useNavigation } from "../hooks2/useNavigation";
import { cn } from "../utils2/cn";
import Button from "./button";

export default function Header() {
  const navigateTo = useNavigation((e) => e.navigateTo);
  const tab = useNavigation((e) => e.current);

  return (
    <header className="border-b border-border flex gap-4 p-3 items-center bg-white sticky top-0">
      <h1 className="text-primary text-xl font-extrabold">ArduSim</h1>
      <span>
        {TABS.map((e) => (
          <a
            onClick={() => navigateTo(e)}
            className={cn(
              `decoration-primary underline-offset-3 decoration-2 text-lg 
              hoverable-gray px-2 py-1 rounded-md cursor-pointer`,
              tab.label === e.label ? "underline" : "hoverable-underline",
            )}
          >
            {e.label}
          </a>
        ))}
      </span>
      <div className="flex-1" />
      <span className="flex flex-row gap-1.5">
        <Button icon="undo" />
        <Button icon="redo" />
        <Button label="Save" type="outlined" />
        <Button label="Start Simulation" type="filled" />
        <Button icon="exit_to_app" />
      </span>
    </header>
  );
}
