import React, { useEffect, useState } from 'react';
import { GetAvailableServices, StartSimulation } from '../wailsjs/go/main/App';
import { DynamicForm } from './components/DynamicForm';
import { ServiceCard } from './components/ServiceCard';
import { UavSidebar } from './components/UavSidebar';
import './FleetConfig.css';
import { uavDisplayName, useAppStore } from './store';

// ─── Helpers ──────────────────────────────────────────────────────────────────

function buildDefaultValues(schemaRaw: string): Record<string, any> {
  try {
    const schema = JSON.parse(schemaRaw);
    if (!schema.properties) return {};
    return Object.fromEntries(
      Object.entries<any>(schema.properties).map(([key, prop]) => [
        key,
        prop.default !== undefined ? prop.default : '',
      ]),
    );
  } catch {
    return {};
  }
}

// ─── Main Component ───────────────────────────────────────────────────────────

const FleetConfig: React.FC = () => {
  const uavs           = useAppStore(s => s.uavs);
  const activeUavId    = useAppStore(s => s.activeUavId);
  const addUavs        = useAppStore(s => s.addUavs);
  const deployService  = useAppStore(s => s.deployService);
  const deleteService  = useAppStore(s => s.deleteService);
  const deployToAll    = useAppStore(s => s.deployToAll);
  const syncAll        = useAppStore(s => s.syncAll);
  const clearAll       = useAppStore(s => s.clearAll);
  const startSim       = useAppStore(s => s.startSimulation);
  const activeMode     = useAppStore(s => s.activeMode);

  const [availableServices, setAvailableServices] = useState<any[]>([]);
  const [selectedServiceId, setSelectedServiceId] = useState<string>('');
  const [formValues,        setFormValues]        = useState<Record<string, any>>({});
  const [editingInstanceId, setEditingInstanceId]  = useState<string | null>(null);
  const [showDeployBox,      setShowDeployBox]      = useState(false);
  const [showAddForm,        setShowAddForm]        = useState(false);
  const [addCount,           setAddCount]           = useState(1);

  useEffect(() => {
    GetAvailableServices().then(res => {
      if (res) {
        setAvailableServices(res);
        if (res.length > 0 && !selectedServiceId) setSelectedServiceId(res[0].id);
      }
    });
  }, []);

  useEffect(() => {
    if (editingInstanceId) return;
    const svc = availableServices.find(s => s.id === selectedServiceId);
    if (svc?.schemaRaw) setFormValues(buildDefaultValues(svc.schemaRaw));
  }, [selectedServiceId, availableServices]);

  const currentUav = uavs.find(u => u.id === activeUavId);
  const selectedService = availableServices.find(s => s.id === selectedServiceId);

  const resetDeployForm = () => {
    setEditingInstanceId(null);
    setShowDeployBox(false);
    const svc = availableServices.find(s => s.id === selectedServiceId);
    if (svc?.schemaRaw) setFormValues(buildDefaultValues(svc.schemaRaw));
  };

  const handleDeploy = () => {
    if (!selectedServiceId) return;
    const payload = {
      serviceId: selectedServiceId,
      serviceTitle: selectedService?.title ?? selectedServiceId,
      config: { ...formValues },
    };
    deployService(activeUavId, payload, editingInstanceId);
    resetDeployForm();
  };

  const handleStartSimulation = async () => {
    try {
      await StartSimulation(uavs as any, activeMode === 'LOCAL');
      startSim(); // Trigger UI tab/state change
    } catch (err) {
      console.error("Failed to start simulation:", err);
      alert("Error starting simulation: " + err);
    }
  };

  if (!currentUav && uavs.length > 0) return null;

  return (
    <div className="fleet-config-container">
      <div className="fleet-content">
        <UavSidebar
          showAddForm={showAddForm}
          setShowAddForm={setShowAddForm}
          addCount={addCount}
          setAddCount={setAddCount}
          onAddSubmit={() => { addUavs(addCount); setShowAddForm(false); setAddCount(1); }}
        />

        <main className="fleet-main">
          {uavs.length > 0 ? (
            <>
              <div className="stack-header">
                <div className="stack-header-left">
                  <span className="stack-title-dot material-symbols-outlined">hub</span>
                  <h2 className="display-font">{uavDisplayName(activeUavId)} Deployment Stack</h2>
                </div>
                <div className="stack-header-actions">
                  <button className="primary-btn" style={{ width: 'auto', marginTop: 0, padding: '0.5rem 1.5rem' }} onClick={handleStartSimulation}>
                    <span className="material-symbols-outlined">play_arrow</span> START SIMULATION
                  </button>
                  <button className="outline-btn" onClick={syncAll}>
                    <span className="material-symbols-outlined">sync</span> Sync All
                  </button>
                  <button className="outline-btn danger-outline" onClick={clearAll}>
                    <span className="material-symbols-outlined">person_remove</span> Clear All
                  </button>
                </div>
              </div>

              {!showDeployBox ? (
                <button className="outline-btn" style={{ marginBottom: '1.5rem', padding: '0.65rem 1.25rem' }}
                  onClick={() => { setShowDeployBox(true); setEditingInstanceId(null); }}>
                  <span className="material-symbols-outlined" style={{ fontSize: '1rem' }}>add</span>
                  Configure New Service
                </button>
              ) : (
                <div className="configure-service-box">
                  <div className="box-header">
                    <h3 className="display-font">Configure New Service</h3>
                    <select className="service-select" value={selectedServiceId}
                      onChange={e => setSelectedServiceId(e.target.value)}>
                      {availableServices.map(s => <option key={s.id} value={s.id}>{s.title}</option>)}
                    </select>
                  </div>

                  {selectedService?.schemaRaw && (
                    <DynamicForm
                      schemaRaw={selectedService.schemaRaw}
                      values={formValues}
                      onChange={(key, val) => setFormValues(prev => ({ ...prev, [key]: val }))}
                    />
                  )}

                  <div className="config-footer">
                    <div className="deploy-actions">
                      <button className="outline-btn" onClick={() => deployToAll({
                        serviceId: selectedServiceId,
                        serviceTitle: selectedService?.title ?? selectedServiceId,
                        config: { ...formValues }
                      })}>Deploy to All</button>
                      <button className="deploy-btn" onClick={handleDeploy}>Deploy to {uavDisplayName(activeUavId)}</button>
                      <button className="close-btn material-symbols-outlined" onClick={resetDeployForm}>close</button>
                    </div>
                  </div>
                </div>
              )}

              {currentUav?.services.length !== 0 && (
                <>
                  <div className="deployed-section-label">Deployed Services</div>
                  <div className="deployed-services-grid">
                    {currentUav!.services.map(svc => (
                      <ServiceCard
                        key={svc.instanceId}
                        service={svc}
                        isEditing={editingInstanceId === svc.instanceId}
                        onEdit={() => {
                          setShowDeployBox(false);
                          setSelectedServiceId(svc.serviceId);
                          setFormValues(svc.config);
                          setEditingInstanceId(svc.instanceId);
                        }}
                        onDelete={() => deleteService(activeUavId, svc.instanceId)}
                        onCancelEdit={resetDeployForm}
                        onUpdate={handleDeploy}
                        renderEditForm={() => {
                          const editSvc = availableServices.find(s => s.id === svc.serviceId);
                          return editSvc?.schemaRaw ? (
                            <DynamicForm
                              schemaRaw={editSvc.schemaRaw}
                              values={formValues}
                              onChange={(key, val) => setFormValues(prev => ({ ...prev, [key]: val }))}
                            />
                          ) : null;
                        }}
                      />
                    ))}
                  </div>
                </>
              )}
            </>
          ) : (
            <div className="stack-header">
              <h2 className="display-font">No UAVs available</h2>
            </div>
          )}
        </main>
      </div>
    </div>
  );
};

export default FleetConfig;
