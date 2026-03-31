import React, { useState } from 'react';
import { ClipboardSetText } from '../wailsjs/runtime/runtime';
import { CheckIcon, CopyIcon, NetworkIcon, TerminalIcon } from './components/Icons';
import { useAppStore } from './store';

export const EnvironmentView: React.FC = () => {
  const activeMode    = useAppStore(s => s.activeMode);
  const masterIP      = useAppStore(s => s.masterIP);
  const setActiveMode = useAppStore(s => s.setActiveMode);
  const setMasterIP   = useAppStore(s => s.setMasterIP);
  const setCurrentTab = useAppStore(s => s.setCurrentTab);

  const [showCommand, setShowCommand] = useState(false);
  const [copied, setCopied]           = useState(false);

  const swarmCommand = `docker swarm join --token SWMTKN-1-49nj... ${masterIP}:2377`;

  const handleEstablish = (e: React.MouseEvent) => {
    e.stopPropagation();
    ClipboardSetText(swarmCommand);
    setShowCommand(true);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
    setCurrentTab('FLEET_CONFIG');
  };

  const handleCopy = (e: React.MouseEvent) => {
    e.stopPropagation();
    ClipboardSetText(swarmCommand);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
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
        <div
          className={`option-card ${activeMode === 'LOCAL' ? 'active-card' : ''}`}
          onClick={() => setActiveMode('LOCAL')}
        >
          <div className="card-header">
            <span className="arch-label">Architecture 01</span>
            <div className="card-icon-badge"><TerminalIcon /></div>
          </div>
          <h3 className="card-title display-font">Local Docker</h3>
          <p className="card-desc">
            Rapid deployment for single-unit testing. Orchestrate simulated
            flight cycles within a sandboxed local container environment.
          </p>
          {activeMode === 'LOCAL' ? (
            <button className="primary-btn display-font"
              onClick={(e) => { e.stopPropagation(); setCurrentTab('FLEET_CONFIG'); }}>
              Establish
            </button>
          ) : (
            <button className="select-btn"
              onClick={(e) => { e.stopPropagation(); setActiveMode('LOCAL'); }}>
              Select Mode →
            </button>
          )}
        </div>

        <div
          className={`option-card ${activeMode === 'SWARM' ? 'active-card' : ''}`}
          onClick={() => setActiveMode('SWARM')}
        >
          <div className="card-header">
            <span className="arch-label">Architecture 02</span>
            <div className="card-icon-badge"><NetworkIcon /></div>
          </div>
          <h3 className="card-title display-font">Docker Swarm</h3>

          {activeMode === 'SWARM' ? (
            <>
              <div className="form-group">
                <label className="label-font">Master Node IP</label>
                <input
                  type="text"
                  value={masterIP}
                  onClick={(e) => e.stopPropagation()}
                  onChange={(e) => setMasterIP(e.target.value)}
                />
              </div>

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
                <button className="primary-btn display-font" onClick={handleEstablish}>
                  Establish
                </button>
              )}
            </>
          ) : (
            <>
              <p className="card-desc">
                Distributed cluster orchestration for high-scale swarm deployment
                across multiple physical or virtual nodes.
              </p>
              <button className="select-btn"
                onClick={(e) => { e.stopPropagation(); setActiveMode('SWARM'); }}>
                Select Mode →
              </button>
            </>
          )}
        </div>
      </div>
    </main>
  );
};
