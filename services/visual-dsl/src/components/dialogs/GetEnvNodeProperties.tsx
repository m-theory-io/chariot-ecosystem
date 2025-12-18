import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface GetEnvNodeProperties {
  varName: string;
}

interface GetEnvNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: GetEnvNodeProperties) => void;
  onDelete: () => void;
  initialProperties: GetEnvNodeProperties;
}

export const GetEnvNodePropertiesDialog: React.FC<GetEnvNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [varName, setVarName] = useState(initialProperties.varName || '');

  const handleSave = () => {
    onSave({
      varName: varName.trim()
    });
    onClose();
  };

  const handleClose = () => {
    onSave({
      varName: varName.trim()
    });
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
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        {/* Title Bar */}
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            GetEnv Properties
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6">
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Environment Variable Name:
            </label>
            <Input
              type="text"
              value={varName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setVarName(e.target.value)}
              className="w-full"
              placeholder="HOME, PATH, DATABASE_URL, etc."
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Name of the environment variable to retrieve (returns DBNull if not set)
            </p>
          </div>

          {/* Action Buttons */}
          <div className="flex gap-3">
            <Button
              onClick={handleClose}
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
