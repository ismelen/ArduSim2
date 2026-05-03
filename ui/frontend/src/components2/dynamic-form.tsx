/* eslint-disable @typescript-eslint/no-explicit-any */
import { useEffect, useState } from "react";
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
  const schema: { properties: Record<string, any> } = JSON.parse(schemaRaw);

  useEffect(() => {
    console.log(currentValues);
    if (Object.entries(currentValues).length !== 0) return;
    if (!schema.properties) return;

    for (const [k, v] of Object.entries(schema.properties)) {
      if (!v.default) continue;
      onChange?.(k, v.default);
    }
  }, []);


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
              key={key}
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
              key={key}
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
                key={key}
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
            key={key}
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
