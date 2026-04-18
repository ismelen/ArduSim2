import { EnvironmentView } from './EnvironmentView';
import FleetConfig from './FleetConfig';
import { GeneralConfigView } from './GeneralConfigView';
import { SimulationView } from './SimulationView';
import './index.css';
import { useNavigation } from './hooks/useNavigation';
import { useConfig } from './hooks/useConfig';
import { Button } from './components/common/Button';

const TABS = ['ENVIRONMENT', 'FLEET_CONFIG', 'GENERAL_CONFIG'] as const;

function App() {
  const { currentTab, setCurrentTab, isSimulating } = useNavigation();
  const { handleStartSimulation, handleLoadSimulation, handleSaveSimulation } = useConfig();


  return (
    <div className="app-container">
      {!isSimulating && (
        <header className="top-nav">
          <div className="nav-group-left">
            <Button
              variant="ghost"
              icon="save"
              onClick={handleSaveSimulation}
              title="SAVE CONFIGURATION"
              style={{ padding: '8px', border: 'none' }}
            />
            <Button
              variant="ghost"
              icon="folder_open"
              onClick={handleLoadSimulation}
              title="LOAD SIMULATION"
              style={{ padding: '8px', border: 'none' }}
            />

            <div className="logo display-font">ArduSim</div>
          </div>

          <nav className="nav-links">
            {TABS.map((tab) => (
              <a
                key={tab}
                href="#"
                className={currentTab === tab ? 'active' : ''}
                onClick={(e) => { e.preventDefault(); setCurrentTab(tab as any); }}
              >
                {tab.replace('_', ' ')}
              </a>
            ))}
          </nav>

          <div className="nav-actions">
            <Button 
              icon="play_circle" 
              onClick={handleStartSimulation}
              style={{ height: '36px', fontSize: '0.7rem' }}
            >
              START SIMULATION
            </Button>
          </div>
        </header>
      )}


      {isSimulating ? (
        <SimulationView />
      ) : (
        <main className="view-wrapper">
          {currentTab === 'ENVIRONMENT' && <EnvironmentView />}
          {currentTab === 'FLEET_CONFIG' && <FleetConfig />}
          {currentTab === 'GENERAL_CONFIG' && <GeneralConfigView />}
        </main>
      )}
    </div>
  );
}

export default App;


