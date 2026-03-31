import { EnvironmentView } from './EnvironmentView';
import FleetConfig from './FleetConfig';
import { SimulationView } from './SimulationView';
import './index.css';
import { useAppStore } from './store';

const TABS = ['ENVIRONMENT', 'FLEET_CONFIG'] as const;

function App() {
  const currentTab    = useAppStore(s => s.currentTab);
  const setCurrentTab = useAppStore(s => s.setCurrentTab);
  const isSimulating  = useAppStore(s => s.isSimulating);

  return (
    <div className="app-container">
      {!isSimulating && (
        <header className="top-nav">
          <div className="logo display-font">ArduSim</div>

          <nav className="nav-links">
            {TABS.map((tab) => (
              <a
                key={tab}
                href="#"
                className={currentTab === tab ? 'active' : ''}
                onClick={(e) => { e.preventDefault(); setCurrentTab(tab); }}
              >
                {tab}
              </a>
            ))}
          </nav>

          {/* Spacer to keep nav centered */}
          <div style={{ width: '80px' }} />
        </header>
      )}

      {isSimulating ? (
        <SimulationView />
      ) : (
        <>
          {currentTab === 'ENVIRONMENT' && <EnvironmentView />}
          {currentTab === 'FLEET_CONFIG' && <FleetConfig />}
        </>
      )}
    </div>
  );
}

export default App;
