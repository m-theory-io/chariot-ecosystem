import React, { useMemo, useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export type BdiNodeProperties = Record<string, string>;

type FieldSpec = {
  key: string;
  label: string;
  placeholder?: string;
  help?: string;
};

interface BdiNodePropertiesProps {
  isOpen: boolean;
  nodeLabel: string;
  onClose: () => void;
  onSave: (properties: BdiNodeProperties) => void;
  onDelete: () => void;
  initialProperties: BdiNodeProperties;
}

const bdiFields: Record<string, FieldSpec[]> = {
  plan: [
    { key: 'name', label: 'Name', placeholder: "'MyPlan'" },
    { key: 'parameters', label: 'Parameters', placeholder: 'array()' },
    { key: 'trigger', label: 'Trigger Function', placeholder: 'func(){ equal(1, 1) }' },
    { key: 'guard', label: 'Guard Function', placeholder: 'func(){ equal(1, 1) }' },
    { key: 'steps', label: 'Steps', placeholder: 'array(stepFn)' },
    { key: 'drop', label: 'Drop Function', placeholder: 'func(){ equal(1, 0) }' }
  ],
  belief: [
    { key: 'agentName', label: 'Agent Name', placeholder: 'thermostat' },
    { key: 'beliefName', label: 'Belief Name', placeholder: 'currentTemp' }
  ],
  agentBelief: [
    { key: 'agentName', label: 'Agent Name', placeholder: 'thermostat' },
    { key: 'beliefName', label: 'Belief Name', placeholder: 'currentTemp' },
    { key: 'value', label: 'Value', placeholder: '72', help: 'Chariot expression or literal value' }
  ],
  agentStartNamed: [
    { key: 'agentName', label: 'Agent Name', placeholder: 'thermostat' },
    { key: 'plan', label: 'Plan', placeholder: 'pThermostat' },
    { key: 'maxConcurrent', label: 'Max Concurrent', placeholder: '1' },
    { key: 'pollSeconds', label: 'Poll Seconds', placeholder: '0' },
    { key: 'lifecycle', label: 'Lifecycle', placeholder: 'eventOnly' }
  ],
  agentStopNamed: [
    { key: 'agentName', label: 'Agent Name', placeholder: 'thermostat' }
  ],
  agentPublish: [
    { key: 'agentName', label: 'Agent Name', placeholder: 'thermostat' }
  ],
  runPlanOnce: [
    { key: 'plan', label: 'Plan', placeholder: 'pThermostat' },
    { key: 'mode', label: 'Mode', placeholder: 'manual' }
  ],
  setStepResult: [
    { key: 'value', label: 'Result Value', placeholder: "map('action', 'cooling_on')", help: 'Chariot expression returned to callers for this step' }
  ],
  setPlanResult: [
    { key: 'value', label: 'Result Value', placeholder: "map('status', 'complete')", help: 'Chariot expression returned to callers for this plan' }
  ],
  signalRegister: [
    { key: 'sourceName', label: 'Source Name', placeholder: 'roomTemp' },
    { key: 'kind', label: 'Kind', placeholder: 'static' },
    { key: 'config', label: 'Config', placeholder: "map('value', 65)", help: 'Examples: map(\'value\', 65), map(\'path\', \'/sys/...\'), map(\'url\', \'https://...\', \'path\', \'data.0.rate\')' }
  ],
  signalRead: [
    { key: 'sourceName', label: 'Source Name', placeholder: 'roomTemp' }
  ],
  signalStartBeliefFeed: [
    { key: 'feedName', label: 'Feed Name', placeholder: 'roomTempFeed' },
    { key: 'sourceName', label: 'Source Name', placeholder: 'roomTemp' },
    { key: 'agentName', label: 'Agent Name', placeholder: 'thermostat' },
    { key: 'beliefName', label: 'Belief Name', placeholder: 'currentTemp' },
    { key: 'intervalSeconds', label: 'Interval Seconds', placeholder: '3' }
  ],
  signalStopBeliefFeed: [
    { key: 'feedName', label: 'Feed Name', placeholder: 'roomTempFeed' }
  ]
};

function fieldsForLabel(label: string): FieldSpec[] {
  return bdiFields[label] || [];
}

export const BdiNodePropertiesDialog: React.FC<BdiNodePropertiesProps> = ({
  isOpen,
  nodeLabel,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const fields = useMemo(() => fieldsForLabel(nodeLabel), [nodeLabel]);
  const [values, setValues] = useState<BdiNodeProperties>(initialProperties || {});

  const handleSave = () => {
    onSave(values);
    onClose();
  };

  const handleDelete = () => {
    onDelete();
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-lg w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {nodeLabel} Properties
          </h3>
          <button
            onClick={onClose}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            x
          </button>
        </div>

        <div className="p-6 space-y-4">
          {fields.length === 0 ? (
            <p className="text-sm text-gray-600 dark:text-gray-300">
              This function has no editable arguments.
            </p>
          ) : fields.map(field => (
            <div key={field.key}>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {field.label}
              </label>
              <Input
                type="text"
                value={values[field.key] || ''}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setValues(prev => ({ ...prev, [field.key]: e.target.value }))}
                className="w-full"
                placeholder={field.placeholder || ''}
              />
              {field.help && (
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{field.help}</p>
              )}
            </div>
          ))}

          <div className="flex gap-3 pt-2">
            <Button
              onClick={handleSave}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Save Properties
            </Button>
            <Button
              onClick={handleDelete}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Delete
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};
