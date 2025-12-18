import React, { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface DefaultNodeProperties {
  name?: string;
  description?: string;
  body?: string;
}

interface DefaultNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: DefaultNodeProperties) => void;
  initialProperties?: DefaultNodeProperties;
}

const defaultState: DefaultNodeProperties = {
  name: '',
  description: '',
  body: ''
};

export const DefaultNodePropertiesDialog: React.FC<DefaultNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {}
}) => {
  const [properties, setProperties] = useState<DefaultNodeProperties>({
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
      description: properties.description?.trim() || '',
      body: properties.body ?? ''
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
          Default Properties
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
              placeholder="Name for this default branch"
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
              placeholder="Describe the fallback behavior"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Inline Statements
            </label>
            <textarea
              value={properties.body || ''}
              onChange={(e) => setProperties(prev => ({ ...prev, body: e.target.value }))}
              placeholder="Statements to execute inside the default block"
              className="w-full h-32 p-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 resize-y"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Paste the statements that should appear inside the default braces.
            </p>
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

export default DefaultNodePropertiesDialog;
