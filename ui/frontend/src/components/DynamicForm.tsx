import React from 'react';
import { Button } from './common/Button';
import { SelectFile } from '../../wailsjs/go/main/App';
import { Switch } from './common/FormField';

interface DynamicFormProps {
  schemaRaw: string;
  values: Record<string, any>;
  onChange: (key: string, value: any) => void;
}

export const DynamicForm: React.FC<DynamicFormProps> = ({ schemaRaw, values, onChange }) => {
  let schema: any;
  try { schema = JSON.parse(schemaRaw); } catch { return null; }
  if (!schema.properties) return null;

  return (
    <div className="dynamic-form-grid horizontal-alignment">
      {Object.entries<any>(schema.properties).map(([key, prop]) => {
        const value = values[key] !== undefined ? values[key] : '';

        if (prop.enum) {
          return (
            <div key={key} className="config-group">
              <label className="label-font">{prop.title || key}</label>
              <div className="service-select-container" style={{ minWidth: '100%' }}>
                <select 
                  className="service-select" 
                  value={value} 
                  onChange={e => onChange(key, e.target.value)}
                  style={{ width: '100%' }}
                >
                  {!value && <option value="" disabled>Select option...</option>}
                  {prop.enum.map((opt: string) => (
                    <option key={opt} value={opt}>{opt.toUpperCase()}</option>
                  ))}
                </select>
              </div>
            </div>
          );
        }

        if (prop.type === 'string' && prop.format === 'kml') {
          return (
            <div key={key} className="config-group">
              <label className="label-font">{prop.title || key}</label>
              <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', width: '100%' }}>
                <input 
                  type="text" 
                  className="service-input" 
                  value={value} 
                  onChange={e => onChange(key, e.target.value)}
                  style={{ flex: 1, minWidth: 0 }} 
                  placeholder="Paste or select route file path..." 
                />
                <Button 
                  variant="outline" 
                  icon="folder_open" 
                  style={{ flexShrink: 0, padding: '0 0.75rem' }}
                  onClick={async () => {
                    try {
                      const path = await SelectFile();
                      if (path) onChange(key, path);
                    } catch (err) {
                      console.error("Failed to select file:", err);
                    }
                  }}
                />
              </div>
            </div>
          );
        }

        if (prop.type === 'boolean') {
          return (
            <div key={key} className="config-group">
              <Switch 
                label={prop.title || key} 
                checked={!!value}
                onChange={checked => onChange(key, checked)}
              />
            </div>
          );
        }

        const isNumeric = prop.type === 'number' || prop.type === 'integer';
        return (
          <div key={key} className="config-group">
            <label className="label-font">{prop.title || key}</label>
            <input
              type={isNumeric ? 'number' : 'text'}
              className="service-input"
              value={value}
              onChange={e => onChange(key, isNumeric ? Number(e.target.value) : e.target.value)}
            />
          </div>
        );
      })}
    </div>
  );
};
