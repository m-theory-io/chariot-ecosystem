import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface Hash256NodeProperties {
  value?: string;
}

interface Hash256NodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: Hash256NodeProperties) => void;
  onDelete: () => void;
  initialProperties: Hash256NodeProperties;
}

const coerceText = (value: unknown, fallback: string) => {
  const text = (value ?? '').toString();
  return text.trim().length > 0 ? text : fallback;
};

export const Hash256NodePropertiesDialog: React.FC<Hash256NodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [value, setValue] = React.useState(() => coerceText(initialProperties.value, 'textValue'));

  React.useEffect(() => {
    setValue(coerceText(initialProperties.value, 'textValue'));
  }, [initialProperties]);

  if (!isOpen) return null;

  const trimmedValue = value.trim();
  const canSave = trimmedValue.length > 0;

  const handleSave = () => {
    if (!canSave) {
      alert('Provide data to hash.');
      return;
    }
    onSave({ value: trimmedValue });
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
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">hash256() Properties</h3>
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
            <code>hash256(data)</code> emits a hex-encoded SHA-256 hash. Supply any literal or expression and the runtime handles byte conversion.
          </p>

          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Data Expression</label>
            <Input
              type="text"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder="payload.body"
              className="w-full"
            />
          </div>

          <div className="flex gap-3 flex-wrap">
            <Button
              onClick={handleSave}
              disabled={!canSave}
              className={`px-6 py-2 ${canSave ? 'bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600' : 'bg-gray-200 dark:bg-gray-600 opacity-60 cursor-not-allowed'} text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200`}
            >
              Save Properties
            </Button>
            <Button
              onClick={handleCancel}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Cancel
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
