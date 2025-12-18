import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface RangeNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: RangeNodeProperties) => void;
  onDelete: () => void;
  initialProperties: RangeNodeProperties;
}

export interface RangeNodeProperties {
  start: string;           // Start value (inclusive)
  end: string;             // End value (exclusive)
}

export const RangeNodePropertiesDialog: React.FC<RangeNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [start, setStart] = React.useState(initialProperties.start || '0');
  const [end, setEnd] = React.useState(initialProperties.end || '10');

  const handleSave = () => {
    onSave({
      start: start.trim(),
      end: end.trim()
    });
    onClose();
  };

  const handleClose = () => {
    // Save properties when closing
    onSave({
      start: start.trim(),
      end: end.trim()
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
            Range Properties
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
          {/* Start Value */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Start Value:
            </label>
            <Input
              type="text"
              value={start}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setStart(e.target.value)}
              className="w-full"
              placeholder="0"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Starting value (inclusive)
            </p>
          </div>

          {/* End Value */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              End Value:
            </label>
            <Input
              type="text"
              value={end}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEnd(e.target.value)}
              className="w-full"
              placeholder="10"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Ending value (exclusive) - range goes from start to end-1
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
