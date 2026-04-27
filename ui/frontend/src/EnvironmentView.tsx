import React from 'react';
import { NetworkIcon, TerminalIcon } from './components/Icons';
import { useEnvironment } from './hooks/useEnvironment';
import { Card } from './components/common/Card';
import { Button } from './components/common/Button';
import { FormField } from './components/common/FormField';

export const EnvironmentView: React.FC = () => {
  const { activeMode, setActiveMode, masterIP, setMasterIP, masterPort, setMasterPort } = useEnvironment();

  return (
    <main className="main-content">
      <div className="env-header">
        <h1 className="env-title display-font">Environment Setup</h1>
        <p className="env-subtitle">
          Define the computational backbone for the mission profile. Select a deployment
          architecture to initialize the UAV neural mesh.
        </p>
      </div>

      <div className="options-grid">
        <Card
          title="LOCAL_NODE"
          subtitle="SINGLE_SIM_ARCHITECTURE"
          headerIcon={<TerminalIcon />}
          active={activeMode === 'LOCAL'}
          onClick={() => setActiveMode('LOCAL')}
        >
          <p className="card-desc">
            Rapid deployment for single-unit testing. Orchestrate simulated
            flight cycles within a sandboxed local container environment.
          </p>
          {activeMode !== 'LOCAL' && (
            <Button variant="select" onClick={(e) => { e.stopPropagation(); setActiveMode('LOCAL'); }}>
              Select Mode →
            </Button>
          )}
        </Card>

        <Card
          title="SWARM_MESH"
          subtitle="DISTRIBUTED_ORCHESTRATOR"
          headerIcon={<NetworkIcon />}
          active={activeMode === 'SWARM'}
          onClick={() => setActiveMode('SWARM')}
        >
          {activeMode === 'SWARM' ? (
            <>
              <FormField label="Master Node IP">
                <input
                  type="text"
                  placeholder="192.168.1.100"
                  value={masterIP}
                  onClick={(e) => e.stopPropagation()}
                  onChange={(e) => setMasterIP(e.target.value)}
                />
              </FormField>
              <FormField label="Docker API Port">
                <input
                  type="number"
                  placeholder="2375"
                  value={masterPort}
                  onClick={(e) => e.stopPropagation()}
                  onChange={(e) => setMasterPort(e.target.value)}
                />
              </FormField>
            </>
          ) : (
            <>
              <p className="card-desc">
                Distributed cluster orchestration for high-scale swarm deployment
                across multiple physical or virtual nodes.
              </p>
              <Button variant="select" onClick={(e) => { e.stopPropagation(); setActiveMode('SWARM'); }}>
                Select Mode →
              </Button>
            </>
          )}
        </Card>
      </div>
    </main>
  );
};
