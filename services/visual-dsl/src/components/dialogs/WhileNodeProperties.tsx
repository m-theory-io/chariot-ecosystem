import React, { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface WhileNodeProperties {
  condition?: string;
  conditionType?: 'expression' | 'variable' | 'comparison';
  leftOperand?: string;
  operator?: 'equal' | 'bigger' | 'smaller' | 'and' | 'or' | 'not' | 'contains' | 'hasPrefix' | 'hasSuffix';
  rightOperand?: string;
  maxIterations?: number;
  description?: string;
  name?: string;
}

interface WhileNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: WhileNodeProperties) => void;
  initialProperties?: WhileNodeProperties;
}

export const WhileNodePropertiesDialog: React.FC<WhileNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {}
}) => {
  const [properties, setProperties] = useState<WhileNodeProperties>({
    condition: '',
    conditionType: 'expression',
    leftOperand: '',
    operator: 'equal',
    rightOperand: '',
    maxIterations: 100,
    description: '',
    name: '',
    ...initialProperties
  });

  useEffect(() => {
    if (isOpen) {
      setProperties({
        condition: '',
        conditionType: 'expression',
        leftOperand: '',
        operator: 'equal',
        rightOperand: '',
        maxIterations: 100,
        description: '',
        name: '',
        ...initialProperties
      });
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
      condition: finalCondition
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
          While Loop Properties
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
              placeholder="Name for this while loop"
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
                Loop Condition
              </label>
              <Input
                type="text"
                value={properties.condition || ''}
                onChange={(e) => setProperties({ ...properties, condition: e.target.value })}
                placeholder="e.g., smaller(i, 10), hasMoreData"
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
                placeholder="e.g., hasMore, isRunning"
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
                  placeholder="e.g., counter, index, length"
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
                  placeholder="e.g., 10, 'end', false"
                />
              </div>
            </>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Max Iterations (safety limit)
            </label>
            <input
              type="number"
              value={String(properties.maxIterations || 100)}
              onChange={(e) => setProperties({ ...properties, maxIterations: parseInt(e.target.value) || 100 })}
              placeholder="100"
              min="1"
              max="10000"
              className="w-full p-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Prevents infinite loops (default: 100)
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Description (optional)
            </label>
            <Input
              type="text"
              value={properties.description || ''}
              onChange={(e) => setProperties({ ...properties, description: e.target.value })}
              placeholder="Describe the purpose of this loop"
            />
          </div>
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

export default WhileNodeProperties;
