import React, { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface IfNodeProperties {
  condition?: string;
  conditionType?: 'expression' | 'variable' | 'comparison';
  leftOperand?: string;
  operator?: 'equal' | 'bigger' | 'smaller' | 'and' | 'or' | 'not' | 'contains' | 'hasPrefix' | 'hasSuffix';
  rightOperand?: string;
  description?: string;
  name?: string;
  ifBody?: string;
  hasElse?: boolean;
  elseBody?: string;
}

interface IfNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: IfNodeProperties) => void;
  initialProperties?: IfNodeProperties;
}

export const IfNodePropertiesDialog: React.FC<IfNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {}
}) => {
  const defaultState: IfNodeProperties = {
    condition: '',
    conditionType: 'expression',
    leftOperand: '',
    operator: 'equal',
    rightOperand: '',
    description: '',
    name: '',
    ifBody: '',
    hasElse: false,
    elseBody: ''
  };

  const coerceBoolean = (value: unknown): boolean => {
    if (typeof value === 'boolean') return value;
    if (typeof value === 'string') {
      const normalized = value.trim().toLowerCase();
      if (normalized === 'true') return true;
      if (normalized === 'false') return false;
    }
    if (typeof value === 'number') {
      return value !== 0;
    }
    return false;
  };

  const hydrateState = (props: IfNodeProperties = {}): IfNodeProperties => {
    const merged = {
      ...defaultState,
      ...props
    };

    const legacyBody = (props as Record<string, unknown>)?.body;
    if ((merged.ifBody === undefined || merged.ifBody === '') && typeof legacyBody === 'string') {
      merged.ifBody = legacyBody;
    }

    merged.ifBody = merged.ifBody ?? '';
    merged.hasElse = coerceBoolean(merged.hasElse);
    merged.elseBody = merged.hasElse ? (merged.elseBody ?? '') : '';

    return merged;
  };

  const [properties, setProperties] = useState<IfNodeProperties>(hydrateState(initialProperties));

  useEffect(() => {
    if (isOpen) {
      setProperties(hydrateState(initialProperties));
    }
  }, [isOpen, initialProperties]);

  const handleSave = () => {
    // Build the condition based on type
    let finalCondition = properties.condition || '';
    
    if (properties.conditionType === 'comparison' && properties.leftOperand && properties.rightOperand) {
      finalCondition = `${properties.operator}(${properties.leftOperand}, ${properties.rightOperand})`;
    } else if (properties.conditionType === 'variable' && properties.leftOperand) {
      finalCondition = properties.leftOperand;
    }

    onSave({
      ...properties,
      condition: finalCondition,
      ifBody: properties.ifBody ?? '',
      hasElse: coerceBoolean(properties.hasElse),
      elseBody: coerceBoolean(properties.hasElse) ? (properties.elseBody ?? '') : ''
    });
    onClose();
  };


  const handleCancel = () => {
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
        <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
          If Statement Properties
        </h3>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Name (optional)
            </label>
            <Input
              type="text"
              value={properties.name || ''}
              onChange={(e) => setProperties({ ...properties, name: e.target.value })}
              placeholder="Name for this If statement"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Condition Type
            </label>
            <select
              value={properties.conditionType}
              onChange={(e) => setProperties({ ...properties, conditionType: e.target.value as any })}
              className="w-full p-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            >
              <option value="expression">Expression</option>
              <option value="variable">Variable</option>
              <option value="comparison">Comparison</option>
            </select>
          </div>

          {properties.conditionType === 'expression' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Condition Expression
              </label>
              <Input
                type="text"
                value={properties.condition || ''}
                onChange={(e) => setProperties({ ...properties, condition: e.target.value })}
                placeholder="e.g., and(bigger(count, 0), equal(status, 'active'))"
              />
            </div>
          )}

          {properties.conditionType === 'variable' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Variable Name
              </label>
              <Input
                type="text"
                value={properties.leftOperand || ''}
                onChange={(e) => setProperties({ ...properties, leftOperand: e.target.value })}
                placeholder="e.g., isValid, hasData"
              />
            </div>
          )}

          {properties.conditionType === 'comparison' && (
            <>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Left Operand
                </label>
                <Input
                  type="text"
                  value={properties.leftOperand || ''}
                  onChange={(e) => setProperties({ ...properties, leftOperand: e.target.value })}
                  placeholder="e.g., age, name, status"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Operator
                </label>
                <select
                  value={properties.operator}
                  onChange={(e) => setProperties({ ...properties, operator: e.target.value as any })}
                  className="w-full p-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                >
                  <option value="equal">equal (equals)</option>
                  <option value="bigger">bigger (greater than)</option>
                  <option value="smaller">smaller (less than)</option>
                  <option value="and">and (logical AND)</option>
                  <option value="or">or (logical OR)</option>
                  <option value="not">not (logical NOT)</option>
                  <option value="contains">contains</option>
                  <option value="hasPrefix">hasPrefix (starts with)</option>
                  <option value="hasSuffix">hasSuffix (ends with)</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Right Operand
                </label>
                <Input
                  type="text"
                  value={properties.rightOperand || ''}
                  onChange={(e) => setProperties({ ...properties, rightOperand: e.target.value })}
                  placeholder="e.g., 18, 'John', true"
                />
              </div>
            </>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Description (optional)
            </label>
            <Input
              type="text"
              value={properties.description || ''}
              onChange={(e) => setProperties({ ...properties, description: e.target.value })}
              placeholder="Describe what this condition checks"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              If Block Statements
            </label>
            <textarea
              value={properties.ifBody || ''}
              onChange={(e) => setProperties({ ...properties, ifBody: e.target.value })}
              placeholder="Statements to execute when the condition is true"
              className="w-full h-32 p-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 resize-y"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Lines entered here render immediately inside the if block, before any nested logicons.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <input
              id="if-has-else"
              type="checkbox"
              checked={properties.hasElse ?? false}
              onChange={(e) => setProperties({ ...properties, hasElse: e.target.checked })}
              className="w-4 h-4 text-blue-600 bg-gray-100 border-gray-300 rounded focus:ring-blue-500 dark:focus:ring-blue-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
            />
            <label htmlFor="if-has-else" className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Include else block
            </label>
          </div>

          {properties.hasElse && (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Else Block Statements
              </label>
              <textarea
                value={properties.elseBody || ''}
                onChange={(e) => setProperties({ ...properties, elseBody: e.target.value })}
                placeholder="Statements to execute when the condition is false"
                className="w-full h-32 p-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 resize-y"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Else statements render after the if block and before any downstream logicons.
              </p>
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 mt-6">
          <Button
            onClick={handleCancel}
            className="px-4 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 text-white"
          >
            Save
          </Button>
        </div>
      </div>
    </div>
  );
};

export default IfNodeProperties;
