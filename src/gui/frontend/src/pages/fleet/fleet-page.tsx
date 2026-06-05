import { useState } from "react";
import { SelectArduPilotInstance } from "../../../wailsjs/go/main/App";
import AddNewServiceDialog from "../../components/add-new-service-dialog";
import Button from "../../components/button";
import Checkbox from "../../components/checkbox";
import FilePickerField from "../../components/file-picker-field";
import FormField from "../../components/form-field";
import ServiceCard from "../../components/service-card";
import { useConfig } from "../../hooks/useConfig";
import { useFleet } from "../../hooks/useFleet";
import { useServices } from "../../hooks/useServices";
import SwarmFormation from "./components/swarm-formation";
import UavsList from "./components/uavs-list";

export default function FleetPage() {
  const uavs = useFleet((s) => s.uavs);
  const deleteService = useFleet((s) => s.deleteService);
  const addService = useFleet((s) => s.addService);
  const updateService = useFleet((s) => s.updateService);
  const updateUav = useFleet((s) => s.updateUav);
  const deleteUav = useFleet((s) => s.deleteUav);
  const activeUavIdx = useFleet((s) => s.activeUavIdx);
  const cloneUav = useFleet((s) => s.cloneUav);
  const { defaultUAVSpeed, defaultArduPilotInstance, defaultMixer, defaultController, batteryCapacity } = useConfig((s) => s.config);
  const { services, mixers, controllers } = useServices();

  const [serviceIdx, setServiceIdx] = useState<number | undefined>(undefined);
  const [baseServiceType, setBaseServiceType] = useState<"mixer" | "controller" | undefined>(undefined);
  const [isConfigCollapsed, setIsConfigCollapsed] = useState(true);

  return (
    <div className="flex h-full">
      <UavsList />
      <div className="flex-1/2 px-3 pt-3 flex flex-col gap-2 pb-5">
        <span className="flex justify-between items-center">
          <h3 className="font-bold text-4xl">Uav {uavs[activeUavIdx].id}</h3>
          <span className="flex gap-2">
            <Button icon="content_copy" type="outlined" onClick={cloneUav} />
            <Button icon="delete" type="outlined" onClick={deleteUav} />
          </span>
        </span>
        <div className="flex flex-col gap-4 mt-4 border border-border p-4 rounded-md shadow-sm">
          <span 
            className="flex justify-between items-center cursor-pointer select-none"
            onClick={() => setIsConfigCollapsed(!isConfigCollapsed)}
          >
            <h4 className="font-semibold text-xl text-dark-gray">UAV Configuration</h4>
            <span className="material-symbols-outlined text-dark-gray">
              {isConfigCollapsed ? "expand_more" : "expand_less"}
            </span>
          </span>
          {!isConfigCollapsed && (
            <div className="flex flex-col gap-2">
            <span className="flex items-end gap-2">
              <div className="flex-1">
                <FormField
                  type="number"
                  label="Speed (m/s)"
                  enabled={uavs[activeUavIdx].speed !== null && uavs[activeUavIdx].speed !== undefined}
                  initValue={`${uavs[activeUavIdx].speed ?? defaultUAVSpeed ?? 10}`}
                  onChange={(val) => updateUav(activeUavIdx, { speed: Number(val) })}
                />
              </div>
              <span className="flex items-center gap-2 mb-2">
                <Checkbox
                  value={uavs[activeUavIdx].speed === null || uavs[activeUavIdx].speed === undefined}
                  onChange={(auto) => updateUav(activeUavIdx, { speed: auto ? null : (defaultUAVSpeed ?? 10) })}
                />
                <label className="text-dark-gray text-sm">Auto</label>
              </span>
            </span>

            <span className="flex items-end gap-2">
              <div className="flex-1">
                <FormField
                  type="number"
                  label="Battery (mAh)"
                  enabled={uavs[activeUavIdx].batteryCapacity !== null && uavs[activeUavIdx].batteryCapacity !== undefined}
                  initValue={`${uavs[activeUavIdx].batteryCapacity ?? batteryCapacity ?? 5000}`}
                  onChange={(val) => updateUav(activeUavIdx, { batteryCapacity: Number(val) })}
                />
              </div>
              <span className="flex items-center gap-2 mb-2">
                <Checkbox
                  value={uavs[activeUavIdx].batteryCapacity === null || uavs[activeUavIdx].batteryCapacity === undefined}
                  onChange={(auto) => updateUav(activeUavIdx, { batteryCapacity: auto ? null : (batteryCapacity ?? 5000) })}
                />
                <label className="text-dark-gray text-sm">Auto</label>
              </span>
            </span>

            <div className="flex flex-col gap-2 mt-2">
              <label className="text-dark-gray">Home Location Override</label>
              <div className="flex gap-2">
                <div className="flex-1">
                  <FormField
                    type="number"
                    label="Latitude"
                    enabled={uavs[activeUavIdx].homeOverride !== null && uavs[activeUavIdx].homeOverride !== undefined}
                    initValue={`${uavs[activeUavIdx].homeOverride?.lat ?? 0}`}
                    onChange={(val) => updateUav(activeUavIdx, { homeOverride: { lat: Number(val), lon: uavs[activeUavIdx].homeOverride?.lon ?? 0 } })}
                  />
                </div>
                <div className="flex-1">
                  <FormField
                    type="number"
                    label="Longitude"
                    enabled={uavs[activeUavIdx].homeOverride !== null && uavs[activeUavIdx].homeOverride !== undefined}
                    initValue={`${uavs[activeUavIdx].homeOverride?.lon ?? 0}`}
                    onChange={(val) => updateUav(activeUavIdx, { homeOverride: { lat: uavs[activeUavIdx].homeOverride?.lat ?? 0, lon: Number(val) } })}
                  />
                </div>
                <span className="flex items-center gap-2 mb-2 self-end">
                  <Checkbox
                    value={uavs[activeUavIdx].homeOverride === null || uavs[activeUavIdx].homeOverride === undefined}
                    onChange={(auto) => updateUav(activeUavIdx, { homeOverride: auto ? null : { lat: 0, lon: 0 } })}
                  />
                  <label className="text-dark-gray text-sm">Auto</label>
                </span>
              </div>
            </div>

            <div className="flex flex-col gap-2 mt-2">
              <span className="flex items-end gap-2">
                <div className="flex-1">
                  <div className={uavs[activeUavIdx].arduPilotInstance === null || uavs[activeUavIdx].arduPilotInstance === undefined ? "opacity-50 pointer-events-none" : ""}>
                    <FilePickerField
                      label="ArduPilot Instance Override"
                      hint="Default SITL instance will be used if empty"
                      initValue={uavs[activeUavIdx].arduPilotInstance ?? defaultArduPilotInstance ?? ""}
                      onChange={(val) => updateUav(activeUavIdx, { arduPilotInstance: val })}
                      onBrowse={SelectArduPilotInstance}
                    />
                  </div>
                </div>
                <span className="flex items-center gap-2 mb-2">
                  <Checkbox
                    value={uavs[activeUavIdx].arduPilotInstance === null || uavs[activeUavIdx].arduPilotInstance === undefined}
                    onChange={(auto) => updateUav(activeUavIdx, { arduPilotInstance: auto ? null : (defaultArduPilotInstance ?? "") })}
                  />
                  <label className="text-dark-gray text-sm">Auto</label>
                </span>
              </span>
            </div>
          </div>
          )}
        </div>
        <span className="flex justify-between items-center mt-5">
          <h5 className="font-semibold text-2xl text-dark-gray">
            Base Services
          </h5>
        </span>
        <ServiceCard
          service={uavs[activeUavIdx].mixer ?? defaultMixer ?? { serviceTitle: "Mixer (Default)" } as any}
          onSelect={() => setBaseServiceType("mixer")}
        />
        <ServiceCard
          service={uavs[activeUavIdx].controller ?? defaultController ?? { serviceTitle: "Controller (Default)" } as any}
          onSelect={() => setBaseServiceType("controller")}
        />

        <span className="flex justify-between items-center mt-5">
          <h5 className="font-semibold text-2xl text-dark-gray">
            Active Services
          </h5>
          <Button
            type="filled"
            icon="add"
            label="Add Service"
            onClick={() => setServiceIdx(-1)}
          />
        </span>
        {uavs[activeUavIdx].services.map((e, idx) => (
          <ServiceCard
            key={idx}
            service={e}
            onDelete={() => deleteService(e)}
            onSelect={() => {
              setServiceIdx(idx);
            }}
          />
        ))}
      </div>
      <SwarmFormation />
      {serviceIdx !== undefined ? (
        <AddNewServiceDialog
          servicesList={services}
          title="Add new service"
          serviceToEdit={
            serviceIdx !== -1
              ? uavs[activeUavIdx].services[serviceIdx]
              : undefined
          }
          onExit={() => setServiceIdx(undefined)}
          onAccept={(service) => {
            if (!service) return setServiceIdx(undefined);

            if (serviceIdx === -1) {
              addService(service);
            } else {
              updateService(serviceIdx, service);
            }
            setServiceIdx(undefined);
          }}
        />
      ) : null}
      {baseServiceType !== undefined ? (
        <AddNewServiceDialog
          servicesList={baseServiceType === "mixer" ? mixers : controllers}
          title={`Edit Base ${baseServiceType === "mixer" ? "Mixer" : "Controller"}`}
          serviceToEdit={
            baseServiceType === "mixer" 
              ? (uavs[activeUavIdx].mixer ?? defaultMixer)
              : (uavs[activeUavIdx].controller ?? defaultController)
          }
          onExit={() => setBaseServiceType(undefined)}
          onAccept={(service) => {
            if (!service) return setBaseServiceType(undefined);

            updateUav(activeUavIdx, {
              [baseServiceType]: service
            });
            setBaseServiceType(undefined);
          }}
        />
      ) : null}
    </div>
  );
}
