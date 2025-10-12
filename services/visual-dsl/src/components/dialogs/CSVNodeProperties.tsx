import React, { useEffect, useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface CSVNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: CSVNodeProperties) => void;
  onDelete: () => void;
  initialProperties: CSVNodeProperties;
}

export interface CSVNodeProperties {
  filename: string;
  delimiter?: string; // optional, default ","
  hasHeaders?: boolean; // optional, default true
}

export const CSVNodePropertiesDialog: React.FC<CSVNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [filename, setFilename] = useState(initialProperties.filename || '');
  const [delimiter, setDelimiter] = useState(initialProperties.delimiter ?? ',');
  const [hasHeaders, setHasHeaders] = useState(initialProperties.hasHeaders ?? true);
  const [canSave, setCanSave] = useState(false);

  useEffect(() => {
    setCanSave(filename.trim().length > 0);
  }, [filename]);

  const handleSave = () => {
    if (!canSave) return;
    onSave({ filename: filename.trim(), delimiter: delimiter, hasHeaders });
    onClose();
  };

  const handleClose = () => {
    if (canSave) {
      onSave({ filename: filename.trim(), delimiter: delimiter, hasHeaders });
    }
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
            CSV Node Properties
          </h3>
          <button
            onClick={handleClose}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {/* Filename (required) */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Filename (required):
            </label>
            <Input
              type="text"
              value={filename}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFilename(e.target.value)}
              className="w-full"
              placeholder="e.g. data/users.csv"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Path is resolved via secure data directory. Only the header row is parsed at this step.
            </p>
          </div>

          {/* Delimiter (optional) */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Delimiter (optional):
            </label>
            <Input
              type="text"
              value={delimiter}
              maxLength={1}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setDelimiter(e.target.value)}
              className="w-24"
              placeholder="," 
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Single character, default is comma. Example: ";" or "\t" for tab.
            </p>
          </div>

          {/* Has Headers toggle (optional) */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Has Headers:
            </label>
            <div className="flex items-center gap-2">
              <input
                id="hasHeaders"
                type="checkbox"
                checked={hasHeaders}
                onChange={(e) => setHasHeaders(e.target.checked)}
                className="h-4 w-4"
              />
              <label htmlFor="hasHeaders" className="text-sm text-gray-700 dark:text-gray-300">
                First row contains column names
              </label>
            </div>
          </div>

          {/* Buttons */}
          <div className="flex gap-3">
            <Button
              onClick={handleSave}
              disabled={!canSave}
              className={`px-6 py-2 ${canSave ? 'bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600' : 'bg-gray-200 dark:bg-gray-600 opacity-60 cursor-not-allowed'} text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200`}
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

      {/* Explanatory text */}
      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          The CSV Node creates a node bound to a CSV file. It parses headers now and lazily loads data later.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. filename (string, required)</li>
          <li>2. delimiter (string, optional, default ",")</li>
          <li>3. hasHeaders (boolean, optional, default true)</li>
        </ul>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          💡 Headers are stored in metadata; the file path is resolved securely at runtime.
        </p>
      </div>
    </div>
  );
};
