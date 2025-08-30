import React, { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface IifNodeProperties {
  condition?: string;
  trueValue?: string;
  falseValue?: string;
  description?: string;
  name?: string;
}

interface IifNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: IifNodeProperties) => void;
  initialProperties?: IifNodeProperties;
}

export const IifNodePropertiesDialog: React.FC<IifNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {}
}) => {
  const [properties, setProperties] = useState<IifNodeProperties>({
    condition: '',
    trueValue: '',
    falseValue: '',
    description: '',
    name: '',
    ...initialProperties
  });

  useEffect(() => {
    if (isOpen) {
      setProperties({
        condition: '',
        trueValue: '',
        falseValue: '',
        description: '',
        name: '',
        ...initialProperties
      });
    }
  }, [isOpen, initialProperties]);

  const handleSave = () => {
    onSave(properties);
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
        <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
          iif Function Properties
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
              placeholder="Name for this iif function"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Condition
            </label>
            <Input
              type="text"
              value={properties.condition || ''}
              onChange={(e) => setProperties({ ...properties, condition: e.target.value })}
              placeholder="e.g., bigger(age, 18)"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Value if True
            </label>
            <Input
              type="text"
              value={properties.trueValue || ''}
              onChange={(e) => setProperties({ ...properties, trueValue: e.target.value })}
              placeholder="Value to return when condition is true"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Value if False
            </label>
            <Input
              type="text"
              value={properties.falseValue || ''}
              onChange={(e) => setProperties({ ...properties, falseValue: e.target.value })}
              placeholder="Value to return when condition is false"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Description (optional)
            </label>
            <Input
              type="text"
              value={properties.description || ''}
              onChange={(e) => setProperties({ ...properties, description: e.target.value })}
              placeholder="Describe what this iif function does"
            />
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-6">
          <Button
            onClick={onClose}
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

export default IifNodeProperties;