interface Props<T> {
  label?: string;
  initValue?: T;
  onChange?(value?: T): void;
  enabled?: boolean;
  options: { value: T; label: string }[];
}

export default function Select<T>({
  initValue,
  onChange,
  label,
  options,
  enabled = true,
}: Props<T>) {
  return (
    <div className="flex gap-2 items-center">
      <label className="text-dark-gray">{label}</label>
      <select
        disabled={!enabled}
        value={initValue as string}
        onChange={(e) => onChange?.(e.target.value as T)}
      >
        {options.map((e) => (
          <option value={e.value as string}>{e.label}</option>
        ))}
      </select>
    </div>
  );
}
