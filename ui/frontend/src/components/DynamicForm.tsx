import React from 'react';

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

        if (prop.type === 'string' && prop.format === 'kml') {
          return (
            <div key={key} className="config-group">
              <label>{prop.title || key}</label>
              <div style={{ display: 'flex', gap: '0.5rem' }}>
                <input
                  type="file"
                  accept=".kml"
                  id={`file-upload-${key}`}
                  style={{ display: 'none' }}
                  onChange={e => {
                    if (e.target.files?.length) onChange(key, e.target.files[0].name);
                  }}
                />
                <input type="text" className="service-input" value={value} readOnly
                  style={{ flex: 1 }} placeholder="Select route file..." />
                <button className="outline-btn" style={{ padding: '0 0.875rem', fontSize: '0.65rem' }}
                  onClick={() => document.getElementById(`file-upload-${key}`)?.click()}>
                  Browse
                </button>
              </div>
            </div>
          );
        }

        if (prop.type === 'boolean') {
          return (
            <label key={key} className="custom-checkbox config-group">
              <input type="checkbox" checked={!!value}
                onChange={e => onChange(key, e.target.checked)} />
              <span className="checkmark" />
              <span className="chk-label">{prop.title || key}</span>
            </label>
          );
        }

        const isNumeric = prop.type === 'number' || prop.type === 'integer';
        return (
          <div key={key} className="config-group">
            <label>{prop.title || key}</label>
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
