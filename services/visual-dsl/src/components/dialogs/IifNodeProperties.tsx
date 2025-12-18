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

  const handleCancel = () => {
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">iif() Properties</h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
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
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
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
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
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
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
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
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
              Description (optional)
            </label>
            <Input
              type="text"
              value={properties.description || ''}
              onChange={(e) => setProperties({ ...properties, description: e.target.value })}
              placeholder="Describe what this iif function does"
            />
          </div>
          <div className="flex gap-3 flex-wrap justify-end pt-2">
            <Button
              onClick={handleSave}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Save Properties
            </Button>
            <Button
              onClick={handleCancel}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Cancel
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default IifNodeProperties;