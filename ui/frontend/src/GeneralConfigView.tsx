import React, { useRef } from 'react';
import { Button } from './components/common/Button';
import { Card } from './components/common/Card';
import { Switch } from './components/common/FormField';
import {
  BarChartIcon,
  FolderIcon,
  SlidersIcon,
  WindIcon
} from './components/Icons';
import { useConfig } from './hooks/useConfig';

export const GeneralConfigView: React.FC = () => {
  const store = useConfig();
  const fileInputRef = useRef<HTMLInputElement>(null);
  
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      store.setSpeedProfilePath(e.target.files[0].name);
    }
  };

  const handleNumericChange = (value: string, setter: (val: number) => void) => {
    if (value === '') {
      setter(0);
    } else {
      const num = parseFloat(value);
      if (!isNaN(num)) setter(num);
    }
  };

  const handleIntegerKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Allow navigation/editing keys
    const allowedKeys = ['Backspace', 'Delete', 'ArrowLeft', 'ArrowRight', 'Tab', 'Enter'];
    if (allowedKeys.includes(e.key)) return;

    // Block non-digit characters
    if (!/^[0-9]$/.test(e.key)) {
      e.preventDefault();
    }
  };

  const handleDecimalKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Allow navigation/editing keys
    const allowedKeys = ['Backspace', 'Delete', 'ArrowLeft', 'ArrowRight', 'Tab', 'Enter', '.'];
    if (allowedKeys.includes(e.key)) {
      // Prevent multiple decimals
      if (e.key === '.' && e.currentTarget.value.includes('.')) {
        e.preventDefault();
      }
      return;
    }

    // Block non-digit characters
    if (!/^[0-9]$/.test(e.key)) {
      e.preventDefault();
    }
  };

  return (
    <main className="main-content">
      <div className="env-header">
        <h1 className="env-title display-font">SYSTEM CONFIGURATION</h1>
        <p className="env-subtitle">
          Define tactical operational parameters for active UAV deployments.
        </p>
      </div>

      <div className="config-layout">
        <div className="config-column">
          <Card 
            title="SIMULATION PARAMETERS" 
            headerIcon={<SlidersIcon />}
            className="config-card"
          >
            <div className="form-group">
              <label className="label-font">SPEED PROFILE (.DAT)</label>
              <div className="file-input-group">
                <input 
                  type="text" 
                  placeholder="PATH/TO/SPEED_PROFILE.DAT" 
                  value={store.speedProfilePath}
                  readOnly
                />
                <input 
                  type="file" 
                  ref={fileInputRef} 
                  style={{ display: 'none' }} 
                  accept=".dat" 
                  onChange={handleFileChange}
                />
                <Button variant="icon" icon={<FolderIcon />} onClick={() => fileInputRef.current?.click()} />
              </div>
            </div>

            <div className="battery-config-row">
              <div className="battery-toggle">
                <span className="label-font">RESTRICT BATTERY</span>
                <label className="switch">
                  <input 
                    type="checkbox" 
                    checked={store.batteryRestricted}
                    onChange={(e) => store.setBatteryRestricted(e.target.checked)}
                  />
                  <span className="slider round"></span>
                </label>
              </div>

              <div className="unit-input" style={{ opacity: store.batteryRestricted ? 1 : 0.5 }}>
                <input 
                  type="number" 
                  placeholder="0"
                  value={store.batteryCapacity || ''}
                  onKeyDown={handleIntegerKeyDown}
                  onChange={(e) => handleNumericChange(e.target.value, store.setBatteryCapacity)}
                  disabled={!store.batteryRestricted}
                  inputMode="numeric"
                  min="0"
                />
                <span className="unit-label">MAH</span>
              </div>
            </div>

            <div style={{ marginTop: '2rem' }}>
              <Switch 
                label="ArduCopter Logging"
                sublabel="ENABLE INTERNAL TELEMETRY RECORDING"
                checked={store.loggingEnabled}
                onChange={store.setLoggingEnabled}
              />
            </div>
          </Card>

          <Card 
            title="GENERAL PARAMETERS" 
            headerIcon={<BarChartIcon />}
            className="config-card"
          >
            <div className="horizontal-toggles">
              <Switch 
                label="Verbose Logging"
                sublabel="RECORD FULL DEBUG STACK TRACES"
                checked={store.verboseLogging}
                onChange={store.setVerboseLogging}
              />
              <Switch 
                label="Store Local Data"
                sublabel="CACHE MISSION HISTORY LOCALLY"
                checked={store.storeLocalData}
                onChange={store.setStoreLocalData}
              />
            </div>
          </Card>
        </div>

        <div className="config-column">
          <Card 
            title="WIND SYSTEM" 
            subtitle="ENV_WIND_VECTOR"
            headerIcon={<WindIcon />}
            className="config-card wind-system-card"
            headerAction={
              <label className="switch">
                <input 
                  type="checkbox" 
                  checked={store.windEnabled}
                  onChange={(e) => store.setWindEnabled(e.target.checked)}
                />
                <span className="slider round"></span>
              </label>
            }
          >
            <div className="centering">
              <div className="wind-compass-container">
                <div className="compass-rim"></div>
                <div className="compass-markers">
                  <span className="marker n">N</span>
                  <span className="marker e">E</span>
                  <span className="marker s">S</span>
                  <span className="marker w">W</span>
                </div>
                <div className="compass-face">
                  <div 
                    className="wind-needle" 
                    style={{ transform: `rotate(${store.windDirection}deg)` }}
                  ></div>
                </div>
              </div>

              <div className="wind-controls" style={{ opacity: store.windEnabled ? 1 : 0.5 }}>
                <div className="form-group">
                  <label className="label-font">DIRECTION</label>
                  <div className="unit-input">
                    <input 
                      type="number" 
                      placeholder="0"
                      value={store.windDirection || ''}
                      onKeyDown={handleIntegerKeyDown}
                      onChange={(e) => handleNumericChange(e.target.value, store.setWindDirection)}
                      disabled={!store.windEnabled}
                      inputMode="numeric"
                      min="0"
                      max="359"
                    />
                    <span className="unit-label">DEG</span>
                  </div>
                </div>

                <div className="form-group">
                  <label className="label-font">WIND SPEED</label>
                  <div className="unit-input">
                    <input 
                      type="number" 
                      placeholder="0.0"
                      value={store.windSpeed || ''}
                      onKeyDown={handleDecimalKeyDown}
                      onChange={(e) => handleNumericChange(e.target.value, store.setWindSpeed)}
                      disabled={!store.windEnabled}
                      inputMode="decimal"
                      step="0.1"
                      min="0"
                    />
                    <span className="unit-label">M/S</span>
                  </div>
                </div>
              </div>
            </div>
          </Card>
          
          <Card 
            title="GROUND FORMATION" 
            subtitle="INITIAL_DEPLOYMENT_LAYOUT"
            headerIcon={<SlidersIcon />}
            className="config-card"
          >
            <div className="form-group">
              <label className="label-font">FORMATION TYPE</label>
              <select 
                className="select-input"
                value={store.groundFormation}
                onChange={(e) => store.setGroundFormation(e.target.value)}
              >
                <option value="LINEAR">LINEAR</option>
                <option value="MATRIX">MATRIX</option>
                <option value="CIRCLE">CIRCLE</option>
                <option value="RANDOM">RANDOM</option>
              </select>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label className="label-font">CENTER LATITUDE</label>
                <div className="unit-input">
                  <input 
                    type="number" 
                    placeholder="39.482594"
                    value={store.formationCenterLat || ''}
                    onKeyDown={handleDecimalKeyDown}
                    onChange={(e) => handleNumericChange(e.target.value, store.setFormationCenterLat)}
                    inputMode="decimal"
                    step="0.000001"
                  />
                  <span className="unit-label">DEG</span>
                </div>
              </div>

              <div className="form-group">
                <label className="label-font">CENTER LONGITUDE</label>
                <div className="unit-input">
                  <input 
                    type="number" 
                    placeholder="-0.346265"
                    value={store.formationCenterLon || ''}
                    onKeyDown={handleDecimalKeyDown}
                    onChange={(e) => handleNumericChange(e.target.value, store.setFormationCenterLon)}
                    inputMode="decimal"
                    step="0.000001"
                  />
                  <span className="unit-label">DEG</span>
                </div>
              </div>
            </div>

            <div className="form-group">
              <label className="label-font">SPACING</label>
              <div className="unit-input">
                <input 
                  type="number" 
                  placeholder="5.0"
                  value={store.formationSpacing || ''}
                  onKeyDown={handleDecimalKeyDown}
                  onChange={(e) => handleNumericChange(e.target.value, store.setFormationSpacing)}
                  inputMode="decimal"
                  step="0.1"
                  min="0"
                />
                <span className="unit-label">METERS</span>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </main>
  );
};


