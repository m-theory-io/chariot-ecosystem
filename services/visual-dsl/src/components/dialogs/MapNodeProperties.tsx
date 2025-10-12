import React, { useEffect, useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface MapNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: MapNodeProperties) => void;
  onDelete: () => void;
  initialProperties: MapNodeProperties;
}

// Backend semantics for mapNode:
// - No args: creates an empty Map node named "map"
// - One string arg: parses map content from the provided string
export interface MapNodeProperties {
  // Optional string content representing a map payload to load
  mapString?: string;
}

export const MapNodePropertiesDialog: React.FC<MapNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [mapString, setMapString] = useState(initialProperties.mapString ?? '');
  const [canSave, setCanSave] = useState(true); // optional field

  useEffect(() => {
    setCanSave(true);
  }, [mapString]);

  const handleSave = () => {
    if (!canSave) return;
    const trimmed = mapString.trim();
    onSave(trimmed ? { mapString: trimmed } : {});
    onClose();
  };

  const handleClose = () => {
    const trimmed = mapString.trim();
    onSave(trimmed ? { mapString: trimmed } : {});
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
            Map Node Properties
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
          {/* Optional map string input */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Map String (optional):
            </label>
            <Input
              type="text"
              value={mapString}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setMapString(e.target.value)}
              className="w-full"
              placeholder="e.g. key1=value1,key2=value2"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Leave blank to create an empty map node named "map". If provided, the backend will parse the string to populate the map.
            </p>
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
          Creates a Map node. With no input, an empty map named "map" is created. If a string is provided, it will be parsed to initialize the map.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. mapString (string, optional)</li>
        </ul>
      </div>
    </div>
  );
};
