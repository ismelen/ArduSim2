import React from "react";
import { ServiceCard } from "./ServiceCard";
import { DynamicForm } from "./DynamicForm";
import { useFleet } from "../hooks/useFleet";

interface ServiceListProps {
  currentUav: any;
  availableServices: any[];
  editingId: string | null;
  startEditing: (svc: any) => void;
  onDelete: (uavId: string, instanceId: string) => void;
  onCancelEdit: () => void;
  onUpdate: () => void;
  formValues: Record<string, any>;
  setFormValues: (values: Record<string, any> | ((prev: any) => any)) => void;
}

export const ServiceList: React.FC<ServiceListProps> = ({
  currentUav,
  availableServices,
  editingId,
  startEditing,
  onDelete,
  onCancelEdit,
  onUpdate,
  formValues,
  setFormValues,
}) => {
  const { activeUavId } = useFleet();

  if (!currentUav?.services || currentUav.services.length === 0) return null;

  return (
    <>
      <div className="deployed-section-label">Deployed Services</div>
      <div className="deployed-services-grid">
        {currentUav.services.map((svc: any) => (
          <ServiceCard
            key={svc.instanceId}
            service={svc}
            isEditing={editingId === svc.instanceId}
            onEdit={() => startEditing(svc)}
            onDelete={() => onDelete(activeUavId, svc.instanceId)}
            onCancelEdit={onCancelEdit}
            onUpdate={onUpdate}
            renderEditForm={() => {
              const editSvc = availableServices.find(
                (s) => s.id === svc.serviceId,
              );
              return editSvc?.schemaRaw ? (
                <DynamicForm
                  schemaRaw={editSvc.schemaRaw}
                  values={formValues}
                  onChange={(key, val) =>
                    setFormValues((prev) => ({ ...prev, [key]: val }))
                  }
                />
              ) : null;
            }}
          />
        ))}
      </div>
    </>
  );
};
