import React, { useMemo, useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface GetTransformNodeProperties {
  transformName: string;
}

interface GetTransformNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: GetTransformNodeProperties) => void;
  onDelete: () => void;
  initialProperties: GetTransformNodeProperties;
  availableTransformNames?: string[];
  transformFetchError?: string;
}

export const GetTransformNodePropertiesDialog: React.FC<GetTransformNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties,
  availableTransformNames = [],
  transformFetchError
}) => {
  const [transformName, setTransformName] = useState(initialProperties.transformName || '');

  const datalistId = useMemo(
    () => `transform-name-options-${Math.random().toString(36).slice(2)}`,
    []
  );

  const normalizedOptions = useMemo(() => {
    const unique = new Set<string>();
    availableTransformNames.forEach((name) => {
      if (typeof name === 'string') {
        const trimmed = name.trim();
        if (trimmed.length > 0) {
          unique.add(trimmed);
        }
      }
    });
    return Array.from(unique).sort((a, b) => a.localeCompare(b));
  }, [availableTransformNames]);

  const canSave = transformName.trim().length > 0;

  const handleSave = () => {
    const nextName = transformName.trim();
    if (!nextName) {
      alert('Transform name is required.');
      return;
    }
    onSave({ transformName: nextName });
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
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-lg w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Get Transform Properties
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Transform Name <span className="text-red-500">*</span>
            </label>
            <Input
              type="text"
              value={transformName}
              onChange={(e) => setTransformName(e.target.value)}
              placeholder="ssn, email, customTransform"
              className="w-full"
              list={normalizedOptions.length > 0 ? datalistId : undefined}
            />
            {normalizedOptions.length > 0 && (
              <datalist id={datalistId}>
                {normalizedOptions.map((option) => (
                  <option value={option} key={option} />
                ))}
              </datalist>
            )}
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Names found in the registry are suggested, but you can type any string or variable.
            </p>
            {transformFetchError && (
              <p className="text-xs text-red-600 dark:text-red-400 mt-1">
                {transformFetchError}
              </p>
            )}
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

      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          <strong>Signature:</strong> getTransform(name)
        </p>
        <p className="text-xs text-gray-600 dark:text-gray-400 mt-2">
          Returns a map that includes description, category, dataType, and the program array for the named transform. You can pass the result into other nodes or use individual fields.
        </p>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          Example:<br />setq(details, getTransform('ssn'))
        </p>
      </div>
    </div>
  );
};
