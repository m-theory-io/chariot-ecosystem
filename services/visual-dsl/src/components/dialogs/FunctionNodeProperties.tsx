import React, { useEffect, useMemo, useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface FunctionParameterEntry {
  name: string;
  value: string;
}

export interface FunctionNodeProperties {
  parameters: FunctionParameterEntry[];
  body: string;
}

interface FunctionNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: FunctionNodeProperties) => void;
  onDelete: () => void;
  initialProperties: FunctionNodeProperties;
}

type ParameterRow = FunctionParameterEntry & { key: string };

const createRow = (name = '', value = ''): ParameterRow => ({
  key: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
  name,
  value,
});

const coerceToRows = (props?: FunctionNodeProperties): ParameterRow[] => {
  const rows: ParameterRow[] = [];
  if (!props) return rows;

  const raw = (props as any).parameters;
  if (Array.isArray(raw)) {
    raw.forEach((entry: any) => {
      if (typeof entry === 'string') {
        const name = entry.toString();
        rows.push(createRow(name, ''));
      } else if (entry && typeof entry === 'object') {
        const name = (entry.name ?? '').toString();
        const value = (entry.value ?? '').toString();
        rows.push(createRow(name, value));
      }
    });
  }

  // Legacy arrays (names/values stored separately)
  const legacyNames = Array.isArray((props as any).parameterNames) ? (props as any).parameterNames : [];
  const legacyValues = Array.isArray((props as any).parameterValues) ? (props as any).parameterValues : [];
  const maxLegacy = Math.max(legacyNames.length, legacyValues.length);
  for (let i = 0; i < maxLegacy; i++) {
    const name = legacyNames[i] ? legacyNames[i].toString() : '';
    const value = legacyValues[i] ? legacyValues[i].toString() : '';
    rows.push(createRow(name, value));
  }

  return rows;
};

const sanitizeParameters = (rows: ParameterRow[]): FunctionParameterEntry[] => {
  return rows
    .map(({ name, value }) => ({ name: name.trim(), value: value.trim() }))
    .filter((entry) => entry.name.length > 0);
};

const FunctionNodePropertiesDialog: React.FC<FunctionNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [parameters, setParameters] = useState<ParameterRow[]>(() => coerceToRows(initialProperties));
  const [body, setBody] = useState<string>(initialProperties.body || '');

  useEffect(() => {
    setParameters(coerceToRows(initialProperties));
    setBody(initialProperties.body || '');
  }, [initialProperties]);

  const preparedParameters = useMemo(() => sanitizeParameters(parameters), [parameters]);
  const bodyTrimmed = useMemo(() => body.trim(), [body]);
  const hasValueWithoutName = useMemo(() => {
    return parameters.some((row) => row.value.trim().length > 0 && row.name.trim().length === 0);
  }, [parameters]);

  const canSave = bodyTrimmed.length > 0 && !hasValueWithoutName;

  const addParameter = () => {
    setParameters((prev) => [...prev, createRow('', '')]);
  };

  const updateParameter = (key: string, field: 'name' | 'value', value: string) => {
    setParameters((prev) => prev.map((row) => (row.key === key ? { ...row, [field]: value } : row)));
  };

  const removeParameter = (key: string) => {
    setParameters((prev) => prev.filter((row) => row.key !== key));
  };

  const persist = () => {
    onSave({
      parameters: preparedParameters,
      body,
    });
  };

  const handleSave = () => {
    if (!canSave) return;
    persist();
    onClose();
  };

  const handleClose = () => {
    if (canSave) {
      persist();
    }
    onClose();
  };



  const handleCancel = () => {
    onClose();
  };
  const handleDelete = () => {
    onDelete();
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-2xl w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Function Properties</h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Parameters (optional):</label>
            <div className="space-y-3">
              {parameters.map((row) => (
                <div key={row.key} className="flex flex-col gap-2 md:flex-row md:items-center md:gap-3">
                  <Input
                    type="text"
                    value={row.name}
                    onChange={(e) => updateParameter(row.key, 'name', e.target.value)}
                    placeholder="name"
                    className="flex-1"
                  />
                  <Input
                    type="text"
                    value={row.value}
                    onChange={(e) => updateParameter(row.key, 'value', e.target.value)}
                    placeholder="value expression (optional)"
                    className="flex-1"
                  />
                  <Button onClick={() => removeParameter(row.key)} className="md:w-auto">Remove</Button>
                </div>
              ))}
              <div>
                <Button onClick={addParameter}>Add Parameter</Button>
              </div>
            </div>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Names define the function signature. Values provide default expressions when callers pass DBNull.</p>
            {hasValueWithoutName && (
              <p className="text-xs text-red-600 dark:text-red-400 mt-1">Add a parameter name for any default value.</p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Body (required):</label>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              className="w-full h-48 p-2 border border-gray-800 dark:border-gray-200 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 rounded"
              placeholder={'// Chariot code\n// Example:\n// return add(x, y)'}
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Enter Chariot code statements. The body will be parsed and stored as a function value.</p>
          </div>

          <div className="flex gap-3">
            <Button onClick={handleSave} disabled={!canSave} className={`px-6 py-2 ${canSave ? 'bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600' : 'bg-gray-200 dark:bg-gray-600 opacity-60 cursor-not-allowed'} text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200`}>Save Properties</Button>
            <Button onClick={handleDelete} className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200">Delete</Button>
          </div>
        </div>
      </div>

      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">Define a function value. You can register it with a name or pass it to call().</p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. parameters (name/value pairs, optional)</li>
          <li>2. body (string, required)</li>
        </ul>
      </div>
    </div>
  );
};

export default FunctionNodePropertiesDialog;
