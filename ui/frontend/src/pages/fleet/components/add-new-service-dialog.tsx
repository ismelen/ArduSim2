/* eslint-disable @typescript-eslint/no-explicit-any */
import { useEffect, useState } from "react";
import { domain } from "../../../../wailsjs/go/models";
import Button from "../../../components/button";
import CardTitle from "../../../components/card-title";
import DynamicForm from "../../../components/dynamic-form";
import { useServices } from "../../../hooks/useServices";
import { cn } from "../../../utils/cn";

interface Props {
  onExit?(): void;
  serviceToEdit?: domain.DeployedService;
  onAccept?(service?: domain.DeployedService): void;
}

export default function AddNewServiceDialog({
  onExit,
  serviceToEdit,
  onAccept,
}: Props) {
  const services = useServices((s) => s.services);
  const [serviceIdx, setServiceIdx] = useState(
    services.findIndex((e) => e.id === serviceToEdit?.serviceId),
  );
  const [currentValues, setCurrentValues] = useState<Record<string, any>>(
    serviceToEdit?.config ?? {},
  );

  useEffect(() => {
    useServices.getState().loadServices();
  }, []);

  return (
    <div className="absolute inset-0 flex h-screen overflow-clip z-60">
      <div
        className="bg-black opacity-60 flex-1/4 cursor-pointer"
        onClick={onExit}
      />
      <div className="bg-background flex-3/4 shadow-lg flex flex-col h-screen">
        <span className="bg-gray border-b border-border p-3 flex justify-between items-center sticky top-0 h-15">
          <CardTitle label="Add new service" icon="add_circle" />
          <Button icon="close" type="outlined" onClick={onExit} />
        </span>
        <div className="bg-red-100 h-100 flex flex-col">
          <div className="bg-blue-100 w-full flex-1">
            {}
          </div>
          <div className="h-15 bg-amber-50 w-full" />
        </div>
        {/* <div className="flex flex-row flex-1 h-[calc(100vh-60px)]">
          <aside className="flex-1/3 h-full border-r border-border max-w-80">
            <h4 className="text-lg font-bold p-3 bg-gray border-b border-border">
              Available Services
            </h4>
            <div className="flex flex-col gap-2 px-2 pt-2 overflow-y-auto">
              {services.map((e, idx) => (
                <ServiceCard
                  service={e}
                  selected={idx === serviceIdx}
                  onSelect={() => setServiceIdx(idx)}
                />
              ))}
            </div>
          </aside>
          <div className="flex-2/3 flex flex-col  overflow-y-auto">
            <div className="bg-background flex-1 p-5 overflow-clip">
              {serviceIdx !== -1 ? (
                <DynamicForm
                  schemaRaw={services[serviceIdx].schemaRaw}
                  values={currentValues}
                  onChange={(key, value) => {
                    setCurrentValues((s) => {
                      s[key] = value;
                      return s;
                    });
                  }}
                />
              ) : null}
              <div className="bg-red-50/50 h-160" />
            </div>
            <div className="flex justify-end gap-3 p-3 border-t border-border bg-gray h-15">
              <Button
                label="Cancel"
                type="outlined"
                onClick={onExit}
                className="px-5 py-2"
              />
              <Button
                label="Accept"
                type="filled"
                className="px-5 py-2"
                onClick={() =>
                  onAccept?.(
                    domain.DeployedService.createFrom({
                      instaceId: serviceToEdit?.instanceId ?? "",
                      serviceId:
                        serviceToEdit?.serviceId ?? services[serviceIdx].id,
                      folderIcon:
                        serviceToEdit?.folderName ??
                        services[serviceIdx].folderName,
                      serviceTitle:
                        serviceToEdit?.serviceTitle ??
                        services[serviceIdx].title,
                      config: currentValues,
                    }),
                  )
                }
              />
            </div>
          </div>
        </div> */}
      </div>
    </div>
  );
}

function ServiceCard({
  service,
  onSelect,
  selected,
}: {
  service: domain.ServiceType;
  onSelect?(): void;
  selected?: boolean;
}) {
  return (
    <div
      onClick={onSelect}
      className={cn(
        `border border-border bg-cwhite shadow-sm rounded-md px-3 py-1.5 
        cursor-pointer hoverable-gray`,
        {
          "bg-primary text-onPrimary hoverable-primary": selected,
        },
      )}
    >
      <p>{service.title}</p>
    </div>
  );
}
