import React from "react";
import { uavDisplayName, useFleet } from "../hooks/useFleet";

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
  const { uavs, activeUavId, setActiveUavId, deleteUav } = useFleet();

  return (
    <aside className="uav-sidebar">
      <div className="sidebar-header">
        <h3 className="display-font label-font">FLEET</h3>
        <button
          className="add-uav-btn material-symbols-outlined"
          onClick={() => setShowAddForm(true)}
        >
          add_circle
        </button>
      </div>

      {showAddForm && (
        <div className="add-uav-popover">
          <label className="label-font">Units to add</label>
          <input
            type="number"
            value={addCount}
            onChange={(e) => setAddCount(parseInt(e.target.value) || 1)}
            min="1"
            max="10"
          />
          <div className="popover-actions">
            <button onClick={() => setShowAddForm(false)}>Cancel</button>
            <button className="confirm-btn" onClick={onAddSubmit}>
              Add
            </button>
          </div>
        </div>
      )}

      <div className="uav-list">
        {uavs.map((uav) => (
          <div
            key={uav.id}
            className={`uav-item ${activeUavId === uav.id ? "active" : ""}`}
            onClick={() => setActiveUavId(uav.id)}
          >
            <div className="uav-designation-label">Designation</div>
            <div className="uav-card-header">
              <h3 className="display-font uav-name">
                {uavDisplayName(uav.id)}
              </h3>
              <button
                className="trash-btn material-symbols-outlined"
                title="Remove UAV"
                onClick={(e) => {
                  e.stopPropagation();
                  deleteUav(uav.id);
                }}
              >
                delete
              </button>
            </div>
            {uav.services.length > 0 && (
              <div className="uav-services-hint">
                {uav.services.map((s) => (
                  <span
                    key={s.instanceId}
                    className="material-symbols-outlined"
                    style={{
                      fontSize: "0.85rem",
                      color: "var(--primary)",
                      opacity: 0.6,
                    }}
                  >
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
