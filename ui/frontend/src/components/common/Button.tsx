import React from "react";

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "outline" | "ghost" | "danger" | "select" | "icon";
  icon?: React.ReactNode;
  children?: React.ReactNode;
}

export const Button: React.FC<ButtonProps> = ({
  variant = "primary",
  icon,
  children,
  className = "",
  ...props
}) => {
  let finalClass = "";
  switch (variant) {
    case "primary":
      finalClass = "primary-btn display-font";
      break;
    case "outline":
      finalClass = "outline-btn";
      break;
    case "ghost":
      finalClass = "ghost-btn";
      break;
    case "danger":
      finalClass = "outline-btn danger-outline";
      break;
    case "select":
      finalClass = "select-btn";
      break;
    case "icon":
      finalClass = "icon-btn";
      break;
    default:
      finalClass = "primary-btn";
  }

  return (
    <button className={`${finalClass} ${className}`} {...props}>
      {icon && (
        <span
          className={
            typeof icon === "string"
              ? "material-symbols-outlined"
              : "button-icon-wrapper"
          }
        >
          {icon}
        </span>
      )}
      {children}
    </button>
  );
};
