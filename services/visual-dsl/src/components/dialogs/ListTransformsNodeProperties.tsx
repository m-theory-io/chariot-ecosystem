import React from 'react';
import { Button } from '../ui/button';

export interface ListTransformsNodeProperties {}

interface ListTransformsNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: ListTransformsNodeProperties) => void;
  onDelete: () => void;
  initialProperties?: ListTransformsNodeProperties;
}

export const ListTransformsNodePropertiesDialog: React.FC<ListTransformsNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties = {}
}) => {
  const handleClose = () => {
    onSave(initialProperties);
    onClose();
  };

  const handleDelete = () => {
    onDelete();
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-lg w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            List Transforms
          </h3>
          <button
            onClick={handleClose}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-4">
          <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
            <strong>listTransforms()</strong> returns an array of transform names currently registered in the runtime. Use
            it when you need to inspect what helpers are available before wiring mappings.
          </p>
          <p className="text-xs text-gray-600 dark:text-gray-400">
            The emitted code is a direct call to <code>listTransforms()</code>. Combine it with <code>setq</code>,
            <code>foreach</code>, or logging nodes to inspect registry contents.
          </p>

          <div className="flex gap-3">
            <Button
              onClick={handleClose}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Close
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

      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          <strong>Example:</strong>
        </p>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          setq(names, listTransforms())<br />logPrint("Transforms:", 'info', names)
        </p>
      </div>
    </div>
  );
};
