import type { ReactNode } from "react";

interface Props {
  hint?: string;
  label?: string;
  type?: React.HTMLInputTypeAttribute;
  initValue?: string;
  onChange?(value: string): void;
  suffix?: ReactNode;
  prefix?: ReactNode;
  min?: number;
  max?: number;
  enabled?: boolean;
}

export default function FormField({
  hint,
  initValue,
  onChange,
  label,
  prefix,
  suffix,
  type,
  min,
  max,
  enabled = true,
}: Props) {
  return (
    <div>
      <label className="text-dark-gray ">{label}</label>
      <span className="flex flex-row items-center gap-2">
        {prefix ?? prefix}
        <input
          disabled={!enabled}
          type={type ?? "text"}
          className="rounded-md border border-border px-3 py-1.5 w-full 
           disabled:opacity-20 placeholder text-field"
          style={{}}
          placeholder={hint}
          value={initValue}
          min={min}
          max={max}
          onChange={(e) => {
            e.preventDefault();
            onChange?.(e.target.value);
          }}
        />
        {suffix ?? suffix}
      </span>
    </div>
  );
}
