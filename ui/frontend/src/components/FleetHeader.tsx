import React from 'react';
import { useFleet, uavDisplayName } from '../hooks/useFleet';
import { Button } from './common/Button';

interface FleetHeaderProps {
  activeUavId: string;
}

export const FleetHeader: React.FC<FleetHeaderProps> = ({ activeUavId }) => {
  const { syncAll, clearAll } = useFleet();

  return (
    <div className="stack-header">
      <div className="stack-header-left" style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
        <span className="material-symbols-outlined" style={{ fontSize: '2rem', color: 'var(--primary)' }}>hub</span>
        <h2 className="env-title display-font" style={{ fontSize: '2.5rem', marginBottom: 0, lineHeight: 1 }}>
          {uavDisplayName(activeUavId)}
        </h2>
      </div>

      <div className="stack-header-actions">
        <div style={{ display: 'flex', gap: '0.75rem' }}>
          <Button variant="outline" icon="sync" onClick={syncAll}>Sync All</Button>
          <Button variant="danger" icon="clear_all" onClick={clearAll}>Clear</Button>
        </div>
      </div>

    </div>
  );

};

