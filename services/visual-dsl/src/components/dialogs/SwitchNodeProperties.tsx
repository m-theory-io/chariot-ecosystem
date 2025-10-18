import React, { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface SwitchNodeProperties {
  name?: string;
  testExpression?: string;
  description?: string;
}

interface SwitchNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: SwitchNodeProperties) => void;
  initialProperties?: SwitchNodeProperties;
}

const defaultState: SwitchNodeProperties = {
  name: '',
  testExpression: '',
  description: ''
};

export const SwitchNodePropertiesDialog: React.FC<SwitchNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {}
}) => {
  const [properties, setProperties] = useState<SwitchNodeProperties>({
    ...defaultState,
    ...initialProperties
  });

  useEffect(() => {
    if (isOpen) {
      setProperties({
        ...defaultState,
        ...initialProperties
      });
    }
  }, [isOpen, initialProperties]);

  const handleSave = () => {
    onSave({
      name: properties.name?.trim() || '',
      testExpression: properties.testExpression?.trim() || '',
      description: properties.description?.trim() || ''
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
          Switch Properties
        </h3>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Name (optional)
            </label>
            <Input
              type="text"
              value={properties.name || ''}
              onChange={(e) => setProperties(prev => ({ ...prev, name: e.target.value }))}
              placeholder="Name for this switch"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Switch Expression
            </label>
            <Input
              type="text"
              value={properties.testExpression || ''}
              onChange={(e) => setProperties(prev => ({ ...prev, testExpression: e.target.value }))}
              placeholder="e.g., getValue(status)"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Description (optional)
            </label>
            <Input
              type="text"
              value={properties.description || ''}
              onChange={(e) => setProperties(prev => ({ ...prev, description: e.target.value }))}
              placeholder="Describe what this switch evaluates"
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

export default SwitchNodePropertiesDialog;
