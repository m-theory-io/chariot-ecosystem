import React, { useEffect, useMemo, useState } from 'react';

export interface AddMappingWithTransformNodeProperties {
  transform: string;
  sourceField: string;
  targetColumn: string;
  transformName: string;
  dataType: string;
  required: boolean;
  defaultValue?: string;
}

interface AddMappingWithTransformNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: AddMappingWithTransformNodeProperties) => void;
  initialProperties?: Partial<AddMappingWithTransformNodeProperties>;
  availableTransformNames?: string[];
  transformFetchError?: string;
}

const dataTypes = ['string', 'int', 'float', 'bool', 'date', 'datetime', 'json'];

const AddMappingWithTransformNodeProperties: React.FC<AddMappingWithTransformNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {},
  availableTransformNames = [],
  transformFetchError
}) => {
  const transformOptions = useMemo(() => {
    const unique = new Set<string>();
    availableTransformNames.forEach((name) => {
      if (typeof name === 'string') {
        const trimmed = name.trim();
        if (trimmed) {
          unique.add(trimmed);
        }
      }
    });
    return Array.from(unique).sort((a, b) => a.localeCompare(b));
  }, [availableTransformNames]);

  const [transform, setTransform] = useState(initialProperties.transform || '');
  const [sourceField, setSourceField] = useState(initialProperties.sourceField || '');
  const [targetColumn, setTargetColumn] = useState(initialProperties.targetColumn || '');
  const normalizedInitialTransformName = (initialProperties.transformName || '').trim();
  const [transformName, setTransformName] = useState(normalizedInitialTransformName);
  const [dataType, setDataType] = useState(initialProperties.dataType || 'string');
  const [required, setRequired] = useState(initialProperties.required || false);
  const [defaultValue, setDefaultValue] = useState(initialProperties.defaultValue || '');
  const [useCustomTransform, setUseCustomTransform] = useState(() => {
    if (!normalizedInitialTransformName) {
      return true;
    }
    return !transformOptions.includes(normalizedInitialTransformName);
  });

  useEffect(() => {
    if (!useCustomTransform && transformName && !transformOptions.includes(transformName)) {
      setUseCustomTransform(true);
    } else if (useCustomTransform && transformName && transformOptions.includes(transformName)) {
      setUseCustomTransform(false);
    }
  }, [useCustomTransform, transformName, transformOptions]);

  if (!isOpen) return null;

  const buildPayload = () => ({
    transform: transform.trim(),
    sourceField: sourceField.trim(),
    targetColumn: targetColumn.trim(),
    transformName: transformName.trim(),
    dataType,
    required,
    defaultValue: defaultValue.trim()
  });

  const validate = () => {
    if (!transform.trim()) {
      alert('Transform variable is required');
      return false;
    }
    if (!sourceField.trim()) {
      alert('Source field is required');
      return false;
    }
    if (!targetColumn.trim()) {
      alert('Target column is required');
      return false;
    }
    if (!transformName.trim()) {
      alert('Transform name is required');
      return false;
    }
    return true;
  };

  const handleSave = () => {
    if (!validate()) {
      return;
    }
    onSave(buildPayload());
    onClose();
  };

  const handleClose = () => {
    if (validate()) {
      onSave(buildPayload());
    }
    onClose();
  };



  const handleCancel = () => {
    onClose();
  };
  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-2xl w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            AddMappingWithTransform Properties
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-300 hover:text-gray-800 dark:hover:text-gray-100"
          >
            ✕
          </button>
        </div>

        <div className="p-4 space-y-4 max-h-[32rem] overflow-y-auto">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Transform Variable <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={transform}
              onChange={(e) => setTransform(e.target.value)}
              placeholder="transformVar"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Naked variable that already holds a Transform instance (e.g. myTransform)
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Source Field <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={sourceField}
              onChange={(e) => setSourceField(e.target.value)}
              placeholder="incoming_field"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Target Column <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={targetColumn}
              onChange={(e) => setTargetColumn(e.target.value)}
              placeholder="target_column"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Transform Name <span className="text-red-500">*</span>
            </label>
            <div className="space-y-2">
              <select
                value={useCustomTransform ? '__custom__' : transformName || '__custom__'}
                onChange={(e) => {
                  if (e.target.value === '__custom__') {
                    setUseCustomTransform(true);
                    if (transformOptions.includes(transformName)) {
                      setTransformName('');
                    }
                  } else {
                    setUseCustomTransform(false);
                    setTransformName(e.target.value);
                  }
                }}
                className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {transformOptions.map((option) => (
                  <option key={option} value={option}>
                    {option}
                  </option>
                ))}
                <option value="__custom__">Custom name...</option>
              </select>
              {useCustomTransform && (
                <input
                  type="text"
                  value={transformName}
                  onChange={(e) => setTransformName(e.target.value)}
                  placeholder={transformOptions.length ? 'Enter or paste transform name' : 'Define transform name'}
                  className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              )}
            </div>
            {transformFetchError && (
              <p className="text-xs text-red-600 dark:text-red-400 mt-1">
                {transformFetchError}
              </p>
            )}
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {transformOptions.length
                ? 'Select a registered transform or choose "Custom name..." to enter a new one.'
                : 'No registered transforms detected yet. Type a name to reference one manually.'}
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Data Type
            </label>
            <select
              value={dataType}
              onChange={(e) => setDataType(e.target.value)}
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {dataTypes.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Default Value (optional)
            </label>
            <input
              type="text"
              value={defaultValue}
              onChange={(e) => setDefaultValue(e.target.value)}
              placeholder="''"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Required Field
            </label>
            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="mapping-required"
                checked={required}
                onChange={(e) => setRequired(e.target.checked)}
                className="w-4 h-4 text-blue-600 bg-gray-100 border-gray-300 rounded focus:ring-blue-500 focus:ring-2"
              />
              <label htmlFor="mapping-required" className="text-sm text-gray-700 dark:text-gray-300">
                This column must be produced
              </label>
            </div>
          </div>

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

      <div className="absolute top-1/2 left-1/2 transform translate-x-96 -translate-y-1/2 max-w-sm p-4 bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          addMappingWithTransform(transform, sourceField, targetColumn, transformName, dataType, required [, defaultValue])
        </p>
        <p className="text-xs text-gray-600 dark:text-gray-400 mt-2 space-y-1">
          <span className="block">1. Transform variable (naked)</span>
          <span className="block">2. Source CSV field</span>
          <span className="block">3. Target column name</span>
          <span className="block">4. Registered transform name</span>
          <span className="block">5. Output data type</span>
          <span className="block">6. Required flag</span>
          <span className="block">7. Optional default value</span>
        </p>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          Example: addMappingWithTransform(myTransform, "price", "unit_price", "moneyToDecimal", "float", true)
        </p>
      </div>
    </div>
  );
};

export default AddMappingWithTransformNodeProperties;
