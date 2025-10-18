import React, { useEffect, useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface SetValueNodeProperties {
  variableName: string;
  value: string;
  valueType?: 'string' | 'expression';
}

interface SetValueNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: SetValueNodeProperties) => void;
  onDelete: () => void;
  initialProperties: Partial<SetValueNodeProperties>;
}

const VALUE_TYPE_OPTIONS: Array<{ value: 'string' | 'expression'; label: string }> = [
  { value: 'string', label: 'String literal' },
  { value: 'expression', label: 'Expression / variable' }
];

const SetValueNodePropertiesDialog: React.FC<SetValueNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const initialVariable = (initialProperties.variableName ?? 'myVar').toString();
  const initialValue = (initialProperties.value ?? '').toString();
  const initialValueType = initialProperties.valueType === 'expression' ? 'expression' : 'string';

  const [variableName, setVariableName] = useState(initialVariable);
  const [value, setValue] = useState(initialValue);
  const [valueType, setValueType] = useState<'string' | 'expression'>(initialValueType);
  const [canSave, setCanSave] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setVariableName(initialVariable);
      setValue(initialValue);
      setValueType(initialValueType);
    }
  }, [isOpen, initialVariable, initialValue, initialValueType]);

  useEffect(() => {
    setCanSave(variableName.trim().length > 0);
  }, [variableName]);

  if (!isOpen) {
    return null;
  }

  const handleSave = () => {
    const trimmedVariable = variableName.trim();
    if (trimmedVariable.length === 0) {
      setCanSave(false);
      return;
    }

    onSave({
      variableName: trimmedVariable,
      value: value.trim(),
      valueType
    });
    onClose();
  };

  const handleDelete = () => {
    onDelete();
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Set Equal Properties</h3>
          <button
            onClick={onClose}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Variable Name<span className="text-red-500 ml-1">*</span></label>
            <Input
              type="text"
              value={variableName}
              onChange={(event) => setVariableName(event.target.value)}
              placeholder="e.g. myVar"
              className="w-full"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Target variable that receives the new value.</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Value</label>
            <Input
              type="text"
              value={value}
              onChange={(event) => setValue(event.target.value)}
              placeholder="Literal text or expression (leave blank to use nested flow)"
              className="w-full"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Provide an inline value, or leave blank and use Create Nesting to build the calculation instead.</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Value Type</label>
            <select
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={valueType}
              onChange={(event) => setValueType(event.target.value as 'string' | 'expression')}
              disabled={value.trim().length === 0}
            >
              {VALUE_TYPE_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Strings are wrapped in quotes. Expressions are emitted exactly (variables, function calls, numbers).
            </p>
          </div>

          <div className="flex gap-3">
            <Button
              onClick={handleSave}
              disabled={!canSave}
              className={`px-6 py-2 ${canSave ? 'bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600' : 'bg-gray-200 dark:bg-gray-600 opacity-60 cursor-not-allowed'} text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200`}
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

      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">Assigns a new value to an existing variable using Chariot's setq.</p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. variableName (target variable)</li>
          <li>2. value (string or expression, optional when using nested logicons)</li>
          <li>3. valueType (string or expression)</li>
        </ul>
      </div>
    </div>
  );
};

export default SetValueNodePropertiesDialog;
