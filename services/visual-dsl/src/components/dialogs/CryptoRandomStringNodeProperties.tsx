import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface CryptoRandomStringNodeProperties {
  length?: string;
}

interface CryptoRandomStringNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: CryptoRandomStringNodeProperties) => void;
  onDelete: () => void;
  initialProperties: CryptoRandomStringNodeProperties;
}

const coerceLength = (value: unknown, fallback: string) => {
  const text = (value ?? '').toString().trim();
  return text.length > 0 ? text : fallback;
};

export const CryptoRandomStringNodePropertiesDialog: React.FC<CryptoRandomStringNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [length, setLength] = React.useState(() => coerceLength(initialProperties.length, 'length'));

  React.useEffect(() => {
    setLength(coerceLength(initialProperties.length, 'length'));
  }, [initialProperties]);

  if (!isOpen) return null;

  const trimmedLength = length.trim();
  const canSave = trimmedLength.length > 0;

  const handleSave = () => {
    if (!canSave) {
      alert('Provide the desired string length.');
      return;
    }
    onSave({ length: trimmedLength });
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
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-xl w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">cryptoRandomString() Properties</h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
            aria-label="Close"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-5">
          <p className="text-sm text-gray-600 dark:text-gray-300">
            <code>cryptoRandomString(length)</code> uses the Crypto Manager to generate a cryptographically secure string. Supply the final length expression (literals or variables) to control the result size.
          </p>

          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Length</label>
            <Input
              type="text"
              value={length}
              onChange={(e) => setLength(e.target.value)}
              placeholder="length"
              className="w-full"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Should evaluate to a positive integer (for example, <code>32</code> or <code>user.tokenSize</code>).</p>
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
