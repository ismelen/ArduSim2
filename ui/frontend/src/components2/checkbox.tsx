interface Props {
  value?: boolean;
  onChange?(value: boolean): void;
}

export default function Checkbox({ value, onChange }: Props) {
  return (
    <input
      type="checkbox"
      value={`${value}`}
      className="custom-checkbox"
      onChange={(e) => onChange?.(e.target.checked)}
    />
  );
}
