import { useState } from "react";
import { SelectFile } from "../../wailsjs/go/main/App";
import Button from "./button";
import FormField from "./form-field";

interface Props {
  hint?: string;
  label?: string;
  initValue?: string;
  onChange?(value: string): void;
}

export default function FilePickerField({
  hint,
  label,
  initValue,
  onChange,
}: Props) {
  const [value, setValue] = useState(initValue ?? "");

  const handleChange = (value: string) => {
    setValue(value);
    onChange?.(value);
  };

  return (
    <FormField
      hint={hint}
      label={label}
      initValue={value}
      onChange={handleChange}
      suffix={
        <Button
          type="outlined"
          icon="folder_open"
          onClick={async () => {
            const file = await SelectFile();
            handleChange(file);
          }}
        />
      }
    />
  );
}
