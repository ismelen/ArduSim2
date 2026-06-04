interface Props {
  icon: string;
  label: string;
}

export default function CardTitle({ icon, label }: Props) {
  return (
    <span className="flex gap-2 items-center">
      <span className="material-symbols-rounded text-primary">{icon}</span>
      <h5 className="font-medium text-xl">{label}</h5>
    </span>
  );
}
