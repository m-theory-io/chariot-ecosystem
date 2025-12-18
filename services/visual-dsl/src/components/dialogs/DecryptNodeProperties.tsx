import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface DecryptNodeProperties {
  keyId?: string;
  ciphertext?: string;
}

interface DecryptNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: DecryptNodeProperties) => void;
  onDelete: () => void;
  initialProperties: DecryptNodeProperties;
}

const coerceText = (value: unknown, fallback: string) => {
  const text = (value ?? '').toString();
  return text.trim().length > 0 ? text : fallback;
};

export const DecryptNodePropertiesDialog: React.FC<DecryptNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [keyId, setKeyId] = React.useState(() => coerceText(initialProperties.keyId, 'encKey'));
  const [ciphertext, setCiphertext] = React.useState(() => coerceText(initialProperties.ciphertext, 'ciphertextBase64'));

  React.useEffect(() => {
    setKeyId(coerceText(initialProperties.keyId, 'encKey'));
    setCiphertext(coerceText(initialProperties.ciphertext, 'ciphertextBase64'));
  }, [initialProperties]);

  if (!isOpen) return null;

  const trimmedKeyId = keyId.trim();
  const trimmedCiphertext = ciphertext.trim();
  const canSave = trimmedKeyId.length > 0 && trimmedCiphertext.length > 0;

  const handleSave = () => {
    if (!canSave) {
      alert('Provide both a key identifier and the base64 ciphertext.');
      return;
    }
    onSave({ keyId: trimmedKeyId, ciphertext: trimmedCiphertext });
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
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">decrypt() Properties</h3>
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
            <code>decrypt(keyID, encryptedData)</code> expects the ciphertext produced by <code>encrypt()</code>. The value must remain base64 encoded until the runtime decodes it, so feed in the literal or expression pointing at that payload.
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
            </div>
            <div>
              <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Ciphertext Expression</label>
              <Input
                type="text"
                value={ciphertext}
                onChange={(e) => setCiphertext(e.target.value)}
                placeholder="payload.ciphertext"
                className="w-full"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Should resolve to the base64 string returned by <code>encrypt()</code> or an external system.</p>
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
