import React from 'react';
import { Button } from '../ui/button';

export interface ContinueNodeProperties {}

interface ContinueNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: ContinueNodeProperties) => void;
  initialProperties?: ContinueNodeProperties;
}

const ContinueNodePropertiesDialog: React.FC<ContinueNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
}) => {
  if (!isOpen) {
    return null;
  }

  const handleDone = () => {
    onSave({});
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4 p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
          Continue Properties
        </h3>
        <p className="text-sm text-gray-700 dark:text-gray-300 mb-6">
          The Continue logicon does not accept any arguments. It emits a <code>continue()</code> statement that skips to the next iteration of the innermost loop when executed.
        </p>
        <div className="flex justify-end">
          <Button
            onClick={handleDone}
            className="px-4 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
          >
            Close
          </Button>
        </div>
      </div>
    </div>
  );
};

export default ContinueNodePropertiesDialog;
