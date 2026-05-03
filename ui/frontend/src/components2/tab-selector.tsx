import { useState, type ReactNode } from "react";
import Button from "./button";

interface TabOption {
  label: string;
  component: ReactNode;
  value: string;
}

interface Props {
  options: TabOption[];
  onChange?(option: TabOption): void;
}

export default function TabSelector({ options, onChange }: Props) {
  const [idx, setIdx] = useState(0);

  return (
    <div>
      <span className="flex gap-1 border border-border rounded-sm p-1">
        {options.map((e, i) => (
          <Button
            label={e.label}
            className="flex-1 items-center justify-center"
            type={idx === i ? "filled" : undefined}
            onClick={() => {
              setIdx(i);
              onChange?.(e);
            }}
          />
        ))}
      </span>
      {options[idx].component}
    </div>
  );
}
