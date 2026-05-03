import React from "react";
import { DynamicForm } from "./DynamicForm";
import { Button } from "./common/Button";
import { uavDisplayName } from "../hooks/useFleet";

interface ServiceDeployerProps {
  availableServices: any[];
  selectedServiceId: string;
  setSelectedServiceId: (id: string) => void;
  selectedService: any;
  formValues: Record<string, any>;
  setFormValues: (values: Record<string, any> | ((prev: any) => any)) => void;
  onDeploy: () => void;
  onDeployToAll: () => void;
  onCancel: () => void;
  activeUavId: string;
}

export const ServiceDeployer: React.FC<ServiceDeployerProps> = ({
  availableServices,
  selectedServiceId,
  setSelectedServiceId,
  selectedService,
  formValues,
  setFormValues,
  onDeploy,
  onDeployToAll,
  onCancel,
  activeUavId,
}) => {
  return (
    <div className="configure-service-box">
      <div className="box-header">
        <h3
          className="display-font"
          style={{
            fontSize: "1.5rem",
            color: "var(--on-surface)",
            display: "flex",
            alignItems: "center",
            gap: "1rem",
            margin: 0,
          }}
        >
          <span
            className="material-symbols-outlined"
            style={{ fontSize: "2.2rem", color: "var(--primary)" }}
          >
            settings_suggest
          </span>
          NEW SERVICE
        </h3>

        <div className="service-select-container">
          <select
            className="service-select"
            value={selectedServiceId}
            onChange={(e) => setSelectedServiceId(e.target.value)}
          >
            {availableServices.map((s) => (
              <option key={s.id} value={s.id}>
                {s.title}
              </option>
            ))}
          </select>
        </div>
      </div>

      {selectedService?.schemaRaw && (
        <DynamicForm
          schemaRaw={selectedService.schemaRaw}
          values={formValues}
          onChange={(key, val) =>
            setFormValues((prev) => ({ ...prev, [key]: val }))
          }
        />
      )}

      <div className="config-footer">
        <div className="deploy-actions">
          <Button variant="outline" icon="sync_alt" onClick={onDeployToAll}>
            Deploy to All
          </Button>
          <Button icon="rocket_launch" onClick={onDeploy}>
            Deploy to {uavDisplayName(activeUavId)}
          </Button>
          <Button variant="icon" icon="close" onClick={onCancel} />
        </div>
      </div>
    </div>
  );
};
