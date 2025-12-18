import React from 'react';
import { Button } from '../ui/button';

export interface TodayNodeProperties {}

interface TodayNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: TodayNodeProperties) => void;
  onDelete: () => void;
  initialProperties: TodayNodeProperties;
}

export const TodayNodePropertiesDialog: React.FC<TodayNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  if (!isOpen) return null;

  const handleSave = () => {
    onSave(initialProperties || {});
    onClose();
  };

  const handleCancel = () => {
    onClose();
  };

  const handleDelete = () => {
    onDelete();
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-lg w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">today() Properties</h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
            aria-label="Close"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-4">
          <p className="text-sm text-gray-600 dark:text-gray-300">
            <code>today()</code> returns the current date in <code>YYYY-MM-DD</code> format and never accepts parameters. Use it anywhere a date literal would normally be required, including comparisons or as the seed for <code>dateAdd()</code>.
          </p>
          <p className="text-sm text-gray-600 dark:text-gray-300">
            Because there are no inputs, this dialog simply confirms that no additional configuration is necessary.
          </p>

          <div className="flex gap-3 flex-wrap">
            <Button
              onClick={handleSave}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Done
            </Button>
            <Button
              onClick={handleDelete}
              className="px-6 py-2 bg-red-100 hover:bg-red-200 dark:bg-red-900/30 dark:hover:bg-red-900/40 text-red-700 dark:text-red-200 border border-red-400 dark:border-red-300"
            >
              Delete Node
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};
