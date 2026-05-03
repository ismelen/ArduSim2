/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState } from "react";
import Checkbox from "./checkbox";
import FilePickerField from "./file-picker-field";
import FormField from "./form-field";
import Select from "./select";

interface Props {
  schemaRaw: string;
  values: Record<string, any>;
  onChange?(key: string, value: any): void;
}

export default function DynamicForm({ schemaRaw, values, onChange }: Props) {
  const [currentValues, setCurrentValues] = useState(values);

  let schema: { properties: Record<string, any> };
  try {
    schema = JSON.parse(schemaRaw);
  } catch {
    return null;
  }

  if (!schema.properties) return null;

  const handleOnChange = (key: string, value: any) => {
    onChange?.(key, value);
    setCurrentValues((s) => ({
      ...s,
      [key]: value,
    }));
  };

  return (
    <div className="flex flex-col gap-2">
      {Object.entries(schema.properties).map(([key, prop]) => {
        if (!prop.title) return null;
        if (prop.enum) {
          return (
            <Select
              initValue={currentValues[key] ?? prop.default}
              options={prop.enum.map((e: any) => ({ label: e, value: e }))}
              label={prop.title || key}
              onChange={(e) => handleOnChange(key, e)}
            />
          );
        }

        if (prop.type === "string" && prop.format) {
          return (
            <FilePickerField
              label={prop.title || key}
              initValue={currentValues[key] ?? prop.default}
              hint={`PATH/TO/FILE.${prop.format}`}
              onChange={(e) => handleOnChange(key, e)}
            />
          );
        }

        if (prop.type === "boolean") {
          return (
            <span className="flex gap-2">
              <Checkbox
                value={currentValues[key] ?? prop.default}
                onChange={(e) => handleOnChange(key, e)}
              />
              <p className="text-dark-gray">{prop.title || key}</p>
            </span>
          );
        }

        const isNumeric = prop.type === "number" || prop.type === "integer";
        return (
          <FormField
            type={isNumeric ? "number" : "text"}
            initValue={currentValues[key] ?? prop.default}
            label={prop.title || key}
            onChange={(e) => handleOnChange(key, e)}
          />
        );
      })}
    </div>
  );
}
