import React from "react";

interface FormFieldProps {
  label: string;
  children: React.ReactNode;
  className?: string;
  sublabel?: string;
  icon?: React.ReactNode;
}

export const FormField: React.FC<FormFieldProps> = ({
  label,
  children,
  className = "",
  sublabel,
  icon,
}) => {
  return (
    <div className={`form-group ${className}`}>
      <label className="label-font">
        {icon && <span className="field-icon">{icon}</span>}
        {label}
      </label>
      {sublabel && <span className="toggle-sublabel">{sublabel}</span>}
      {children}
    </div>
  );
};

interface SwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  sublabel?: string;
}

export const Switch: React.FC<SwitchProps> = ({
  checked,
  onChange,
  label,
  sublabel,
}) => {
  return (
    <div className="toggle-list-item">
      <div className="toggle-info">
        <span className="toggle-label">{label}</span>
        {sublabel && <span className="toggle-sublabel">{sublabel}</span>}
      </div>
      <label className="switch">
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
        />
        <span className="slider round"></span>
      </label>
    </div>
  );
};
