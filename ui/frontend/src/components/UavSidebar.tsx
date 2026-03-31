import React from 'react';
import { uavDisplayName, useAppStore } from '../store';

interface UavSidebarProps {
  showAddForm: boolean;
  setShowAddForm: (show: boolean) => void;
  addCount: number;
  setAddCount: (count: number) => void;
  onAddSubmit: () => void;
}

export const UavSidebar: React.FC<UavSidebarProps> = ({
  showAddForm,
  setShowAddForm,
  addCount,
  setAddCount,
  onAddSubmit,
}) => {
  const uavs = useAppStore(s => s.uavs);
  const activeUavId = useAppStore(s => s.activeUavId);
  const setActiveUavId = useAppStore(s => s.setActiveUavId);
  const deleteUav = useAppStore(s => s.deleteUav);

  return (
    <aside className="fleet-sidebar">
      <div className="sidebar-heading">
        <span className="sidebar-heading-title">Unit_Inventory</span>
        <span className="sidebar-total-badge">Total: {uavs.length.toString().padStart(2, '0')}</span>
      </div>

      {showAddForm ? (
        <div className="add-uav-form">
          <div className="add-uav-label">UAVs to add</div>
          <input
            type="number"
            min={1}
            value={addCount}
            onChange={e => setAddCount(parseInt(e.target.value) || 1)}
            className="service-input add-uav-input"
          />
          <div className="add-uav-actions">
            <button className="add-uav-action-btn add-uav-action-btn--primary" onClick={onAddSubmit}>
              Add
            </button>
            <button className="add-uav-action-btn add-uav-action-btn--danger" onClick={() => setShowAddForm(false)}>
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <button className="add-uav-btn" onClick={() => setShowAddForm(true)}>
          <span className="material-symbols-outlined" style={{ fontSize: '1.1rem' }}>add</span>
          Add UAV
        </button>
      )}

      <div className="uav-list">
        {uavs.map(uav => (
          <div
            key={uav.id}
            className={`uav-card ${activeUavId === uav.id ? 'active-uav' : ''}`}
            onClick={() => setActiveUavId(uav.id)}
          >
            <div className="uav-designation-label">Designation</div>
            <div className="uav-card-header">
              <h3 className="display-font uav-name">{uavDisplayName(uav.id)}</h3>
              <button className="trash-btn material-symbols-outlined" title="Remove UAV"
                onClick={e => { e.stopPropagation(); deleteUav(uav.id); }}>
                delete
              </button>
            </div>
            {uav.services.length > 0 && (
              <div className="uav-services-hint">
                {uav.services.map(s => (
                  <span key={s.instanceId} className="material-symbols-outlined"
                    style={{ fontSize: '0.9rem', color: 'var(--primary)', opacity: 0.7 }}>
                    memory
                  </span>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </aside>
  );
};
