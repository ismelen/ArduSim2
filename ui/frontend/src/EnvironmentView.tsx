import React, { useState } from 'react';
import { ClipboardSetText } from '../wailsjs/runtime/runtime';
import { CheckIcon, CopyIcon, NetworkIcon, TerminalIcon } from './components/Icons';
import { useEnvironment } from './hooks/useEnvironment';
import { useNavigation } from './hooks/useNavigation';
import { Card } from './components/common/Card';
import { Button } from './components/common/Button';
import { FormField } from './components/common/FormField';

export const EnvironmentView: React.FC = () => {
  const { activeMode, setActiveMode, masterIP, setMasterIP, showCommand, setShowCommand } = useEnvironment();
  const { setCurrentTab } = useNavigation();

  const [copied, setCopied] = useState(false);

  const swarmCommand = `docker swarm join --token SWMTKN-1-49nj... ${masterIP}:2377`;


  const handleCopy = (e: React.MouseEvent) => {
    e.stopPropagation();
    ClipboardSetText(swarmCommand);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleEstablish = (e: React.MouseEvent) => {

    e.stopPropagation();
    ClipboardSetText(swarmCommand);
    setShowCommand(true);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
    // Wait 1s then navigate to FLEET_CONFIG
    setTimeout(() => {
      setCurrentTab('FLEET_CONFIG');
    }, 1000);
  };

  const handleLocalSelect = () => {
    setActiveMode('LOCAL');
    setCurrentTab('FLEET_CONFIG');
  };


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
          onClick={handleLocalSelect}
        >
          <p className="card-desc">
            Rapid deployment for single-unit testing. Orchestrate simulated
            flight cycles within a sandboxed local container environment.
          </p>
          {activeMode !== 'LOCAL' && (
            <Button variant="select" onClick={(e) => { e.stopPropagation(); handleLocalSelect(); }}>
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
                  value={masterIP}
                  onClick={(e) => e.stopPropagation()}
                  onChange={(e) => setMasterIP(e.target.value)}
                />
              </FormField>

              {showCommand ? (
                <div className="command-box">
                  <span className="command-box-label">Run on worker nodes:</span>
                  <div className="command-box-content">
                    <code>{swarmCommand}</code>
                    <button className="copy-btn" onClick={handleCopy} title="Copy">
                      {copied ? <CheckIcon /> : <CopyIcon />}
                    </button>
                  </div>
                </div>
              ) : (
                <Button onClick={handleEstablish}>Establish</Button>
              )}
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

