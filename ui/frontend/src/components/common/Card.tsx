import React from "react";

interface CardProps {
  children?: React.ReactNode;
  title?: string | React.ReactNode;
  subtitle?: string;
  className?: string;
  headerAction?: React.ReactNode;
  headerIcon?: React.ReactNode;
  active?: boolean;
  onClick?: () => void;
  footer?: React.ReactNode;
}

export const Card: React.FC<CardProps> = ({
  children,
  title,
  subtitle,
  className = "",
  headerAction,
  headerIcon,
  active,
  onClick,
  footer,
}) => {
  return (
    <div
      className={`option-card ${active ? "active-card" : ""} ${className}`}
      onClick={onClick}
    >
      {(title || headerIcon || headerAction) && (
        <div className="card-header">
          <div className="card-title-group">
            {headerIcon && <div className="card-icon-badge">{headerIcon}</div>}
            <div className="card-text-group">
              {title && <h3 className="card-title display-font">{title}</h3>}
              {subtitle && <span className="arch-label">{subtitle}</span>}
            </div>
          </div>
          {headerAction && <div className="header-action">{headerAction}</div>}
        </div>
      )}
      <div className="card-body">{children}</div>
      {footer && <div className="card-footer">{footer}</div>}
    </div>
  );
};
