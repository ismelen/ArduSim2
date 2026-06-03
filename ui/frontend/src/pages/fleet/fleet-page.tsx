import { useState } from "react";
import Button from "../../components/button";
import { useFleet } from "../../hooks/useFleet";
import AddNewServiceDialog from "./components/add-new-service-dialog";
import SwarmFormation from "./components/swarm-formation";
import UavsList from "./components/uavs-list";
import ServiceCard from "./components/service-card";

export default function FleetPage() {
  const uavs = useFleet((s) => s.uavs);
  const deleteService = useFleet((s) => s.deleteService);
  const addService = useFleet((s) => s.addService);
  const updateService = useFleet((s) => s.updateService);
  const deleteUav = useFleet((s) => s.deleteUav);
  const activeUavIdx = useFleet((s) => s.activeUavIdx);
  const cloneUav = useFleet((s) => s.cloneUav);

  const [serviceIdx, setServiceIdx] = useState<number | undefined>(undefined);

  return (
    <div className="flex h-full">
      <UavsList />
      <div className="flex-1/2 px-3 pt-3 flex flex-col gap-2">
        <span className="flex justify-between items-center">
          <h3 className="font-bold text-4xl">Uav {uavs[activeUavIdx].id}</h3>
          <span className="flex gap-2">
            <Button icon="content_copy" type="outlined" onClick={cloneUav} />
            <Button icon="delete" type="outlined" onClick={deleteUav} />
          </span>
        </span>
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
          serviceToEdit={
            serviceIdx !== -1
              ? uavs[activeUavIdx].services[serviceIdx]
              : undefined
          }
          onExit={() => setServiceIdx(undefined)}
          onAccept={(service) => {
            console.log(service);
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
    </div>
  );
}
