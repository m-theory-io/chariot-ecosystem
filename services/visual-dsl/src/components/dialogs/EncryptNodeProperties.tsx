import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface EncryptNodeProperties {
  keyId?: string;
  data?: string;
}

interface EncryptNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: EncryptNodeProperties) => void;
  onDelete: () => void;
  initialProperties: EncryptNodeProperties;
}

const coerceText = (value: unknown, fallback: string) => {
  const text = (value ?? '').toString();
  return text.trim().length > 0 ? text : fallback;
};

export const EncryptNodePropertiesDialog: React.FC<EncryptNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [keyId, setKeyId] = React.useState(() => coerceText(initialProperties.keyId, 'encKey'));
  const [data, setData] = React.useState(() => coerceText(initialProperties.data, 'plaintextValue'));

  React.useEffect(() => {
    setKeyId(coerceText(initialProperties.keyId, 'encKey'));
    setData(coerceText(initialProperties.data, 'plaintextValue'));
  }, [initialProperties]);

  if (!isOpen) return null;

  const trimmedKeyId = keyId.trim();
  const trimmedData = data.trim();
  const canSave = trimmedKeyId.length > 0 && trimmedData.length > 0;

  const handleSave = () => {
    if (!canSave) {
      alert('Provide both a key identifier and data to encrypt.');
      return;
    }
    onSave({ keyId: trimmedKeyId, data: trimmedData });
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
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-2xl w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">encrypt() Properties</h3>
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
            <code>encrypt(keyID, data)</code> looks up the managed key, encrypts your payload, and returns a base64 ciphertext. Provide the key identifier exactly as configured in the vault and the expression that resolves to the plaintext.
          </p>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Key ID</label>
              <Input
                type="text"
                value={keyId}
                onChange={(e) => setKeyId(e.target.value)}
                placeholder="encKey"
                className="w-full"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Matches the secure key registered with the Crypto Manager.</p>
            </div>
            <div>
              <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Plaintext Expression</label>
              <Input
                type="text"
                value={data}
                onChange={(e) => setData(e.target.value)}
                placeholder="user.secret"
                className="w-full"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Can be a literal or any expression that evaluates to a string.</p>
            </div>
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
