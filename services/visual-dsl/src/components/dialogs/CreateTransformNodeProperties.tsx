import React, { useState } from 'react';

export interface CreateTransformNodeProperties {
  transformName: string;
}

interface CreateTransformNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: CreateTransformNodeProperties) => void;
  initialProperties?: Partial<CreateTransformNodeProperties>;
}

const CreateTransformNodeProperties: React.FC<CreateTransformNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {}
}) => {
  const [transformName, setTransformName] = useState(initialProperties.transformName || '');

  const handleSave = () => {
    if (!transformName.trim()) {
      alert('Transform name is required');
      return;
    }

    onSave({
      transformName: transformName.trim()
    });
    onClose();
  };

  const handleClose = () => {
    // Save properties when closing
    if (transformName.trim()) {
      onSave({
        transformName: transformName.trim()
      });
    }
    onClose();
  };



  const handleCancel = () => {
    onClose();
  };
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        {/* Title Bar */}
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            CreateTransform Properties
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-300 hover:text-gray-800 dark:hover:text-gray-100"
          >
            ✕
          </button>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4">
          {/* Transform Name Input */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Transform Name <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={transformName}
              onChange={(e) => setTransformName(e.target.value)}
              placeholder="Enter transform name"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Use variable names (e.g. myTransform) for variables, or text that will be quoted as strings
            </p>
          </div>

          {/* Buttons */}
          <div className="flex gap-3 pt-2">
            <button
              onClick={handleSave}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Save Properties
            </button>
            <button
              onClick={handleCancel}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
      
      {/* Explanatory text */}
      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          The CreateTransform logicon creates a new transform object that can be used to transform data between different formats.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. Transform name (required, string or variable)</li>
        </ul>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          💡 Examples:<br/>
          createTransform("userTransform") → creates named transform<br/>
          createTransform(transformVar) → creates transform using variable name<br/>
          createTransform("dataMapper") → creates data mapping transform
        </p>
      </div>
    </div>
  );
};

export default CreateTransformNodeProperties;
