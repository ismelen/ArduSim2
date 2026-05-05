import { TABS, useNavigation } from "../hooks/useNavigation";
import { useSimulation } from "../hooks/useSimulation";
import { useTheme } from "../hooks/useTheme";
import { cn } from "../utils/cn";
import Button from "./button";

export default function Header() {
  const navigateTo = useNavigation((e) => e.navigateTo);
  const tab = useNavigation((e) => e.current);

  const lastConfig = useSimulation((s) => s.lastConfig);
  const undos = useSimulation((s) => s.undos);
  const redos = useSimulation((s) => s.redos);
  const lastHash = useSimulation((s) => s.lastHash);
  const undo = useSimulation((s) => s.undo);
  const redo = useSimulation((s) => s.redo);
  const saveConfig = useSimulation((s) => s.saveConfig);
  const newConfig = useSimulation((s) => s.newConfig);

  const isNight = useTheme((s) => s.isNight);
  const toggleTheme = useTheme((s) => s.toggleTheme);

  return (
    <header className="border-b border-border flex gap-4 p-3 items-center bg-cwhite sticky top-0">
      <Button icon="note_add" onClick={newConfig} type="filled" />
      <h1 className="text-primary text-xl font-extrabold">ArduSim</h1>
      <span>
        {TABS.map((e) => (
          <a
            key={e.label}
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
        <Button icon="undo" onClick={undo} enabled={undos.length !== 0} />
        <Button icon="redo" onClick={redo} enabled={redos.length !== 0} />
        <Button
          label="Save"
          type="outlined"
          onClick={saveConfig}
          enabled={lastConfig.hash !== lastHash}
        />
        <Button label="Start Simulation" type="filled" />
        <Button icon={isNight ? "bedtime" : "sunny"} onClick={toggleTheme} />
      </span>
    </header>
  );
}
