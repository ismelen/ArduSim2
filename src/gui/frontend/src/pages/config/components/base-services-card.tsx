import { useState } from "react";

import ServiceCard from "../../../components/service-card";
import AddNewServiceDialog from "../../../components/add-new-service-dialog";
import { useConfig } from "../../../hooks/useConfig";
import { useServices } from "../../../hooks/useServices";

export default function BaseServicesCard() {
  const config = useConfig((s) => s.config);
  const updateConfig = useConfig((s) => s.update);
  const { mixers } = useServices();

  const [editType, setEditType] = useState<"mixer" | undefined>(undefined);

  return (
    <div className="border border-border p-4 rounded-md shadow-sm">
      <h4 className="font-semibold text-xl text-dark-gray mb-3">Base Services Defaults</h4>
      
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-dark-gray text-sm">Default Mixer</label>
          <ServiceCard
            service={config.defaultMixer ?? { serviceTitle: "Select Default Mixer" } as any}
            onSelect={() => setEditType("mixer")}
          />
        </div>
      </div>

      {editType !== undefined && (
        <AddNewServiceDialog
          servicesList={mixers}
          title={`Edit Default Mixer`}
          serviceToEdit={config.defaultMixer}
          onExit={() => setEditType(undefined)}
          onAccept={(service) => {
            if (!service) return setEditType(undefined);
            updateConfig((c) => ({
              ...c,
              defaultMixer: service
            }));
            setEditType(undefined);
          }}
        />
      )}
    </div>
  );
}
