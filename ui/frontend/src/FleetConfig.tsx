import React, { useEffect } from 'react';
import { UavSidebar } from './components/UavSidebar';
import { FleetHeader } from './components/FleetHeader';
import { ServiceDeployer } from './components/ServiceDeployer';
import { ServiceList } from './components/ServiceList';
import { Button } from './components/common/Button';
import './FleetConfig.css';
import { useFleet } from './hooks/useFleet';
import { useServices } from './hooks/useServices';

const FleetConfig: React.FC = () => {
  const fleet = useFleet();
  const { fetchServices } = useServices();

  const [showAddForm, setShowAddForm] = React.useState(false);
  const [addCount, setAddCount] = React.useState(1);

  useEffect(() => {
    fetchServices();
  }, [fetchServices]);

  const currentUav = fleet.uavs.find(u => u.id === fleet.activeUavId);

  if (!currentUav && fleet.uavs.length > 0) return null;

  return (
    <div className="fleet-config-container">
      <div className="fleet-content">
        <UavSidebar
          showAddForm={showAddForm}
          setShowAddForm={setShowAddForm}
          addCount={addCount}
          setAddCount={setAddCount}
          onAddSubmit={() => { fleet.addUavs(addCount); setShowAddForm(false); setAddCount(1); }}
        />

        <main className="fleet-main">
          {fleet.uavs.length > 0 ? (
            <>
              <FleetHeader activeUavId={fleet.activeUavId} />

              {!fleet.showDeployBox ? (
                <div style={{ display: 'flex', justifyContent: 'flex-start' }}>
                  <Button 
                    variant="outline" 
                    icon="add" 
                    style={{ marginBottom: '2.5rem', padding: '0.8rem 1.5rem' }}
                    onClick={() => fleet.setShowDeployBox(true)}
                  >
                    Configure New Service
                  </Button>
                </div>
              ) : (
                <ServiceDeployer
                  availableServices={useServices.getState().availableServices}
                  selectedServiceId={fleet.selectedServiceId}
                  setSelectedServiceId={fleet.setSelectedServiceId}
                  selectedService={useServices.getState().availableServices.find(s => s.id === fleet.selectedServiceId)}
                  formValues={fleet.formValues}
                  setFormValues={fleet.setFormValues}
                  onDeploy={fleet.handleDeploy}
                  onDeployToAll={fleet.handleDeployToAll}
                  onCancel={fleet.resetDeployForm}
                  activeUavId={fleet.activeUavId}
                />
              )}

              <ServiceList
                currentUav={currentUav}
                availableServices={useServices.getState().availableServices}
                editingId={fleet.editingInstanceId}
                startEditing={fleet.startEditing}
                onDelete={fleet.deleteService}
                onCancelEdit={fleet.resetDeployForm}
                onUpdate={fleet.handleDeploy}
                formValues={fleet.formValues}
                setFormValues={fleet.setFormValues}
              />
            </>
          ) : (
            <div className="env-header">
              <h1 className="env-title display-font">No UAVs Registered</h1>
              <p className="env-subtitle">Initialize your fleet from the registry sidebar to begin mission planning.</p>
            </div>
          )}
        </main>
      </div>
    </div>
  );

};


export default FleetConfig;
