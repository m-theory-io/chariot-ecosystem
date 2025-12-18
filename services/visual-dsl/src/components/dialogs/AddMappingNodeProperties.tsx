import React, { useState } from 'react';

export interface AddMappingNodeProperties {
  transform: string;
  sourceField: string;
  targetColumn: string;
  program: string[];
  dataType: string;
  required: boolean;
}

interface AddMappingNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: AddMappingNodeProperties) => void;
  initialProperties?: Partial<AddMappingNodeProperties>;
}

const AddMappingNodeProperties: React.FC<AddMappingNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {}
}) => {
  const [transform, setTransform] = useState(initialProperties.transform || '');
  const [sourceField, setSourceField] = useState(initialProperties.sourceField || '');
  const [targetColumn, setTargetColumn] = useState(initialProperties.targetColumn || '');
  const [program, setProgram] = useState<string[]>(initialProperties.program || []);
  const [dataType, setDataType] = useState(initialProperties.dataType || 'string');
  const [required, setRequired] = useState(initialProperties.required || false);
  const [newProgramLine, setNewProgramLine] = useState('');

  const dataTypes = ['string', 'int', 'float', 'bool', 'date', 'datetime', 'json'];

  const handleSave = () => {
    if (!transform.trim()) {
      alert('Transform variable is required');
      return;
    }
    if (!sourceField.trim()) {
      alert('Source field is required');
      return;
    }
    if (!targetColumn.trim()) {
      alert('Target column is required');
      return;
    }

    onSave({
      transform: transform.trim(),
      sourceField: sourceField.trim(),
      targetColumn: targetColumn.trim(),
      program: program.filter(line => line.trim() !== ''),
      dataType,
      required
    });
    onClose();
  };

  const handleClose = () => {
    // Save properties when closing if required fields are filled
    if (transform.trim() && sourceField.trim() && targetColumn.trim()) {
      onSave({
        transform: transform.trim(),
        sourceField: sourceField.trim(),
        targetColumn: targetColumn.trim(),
        program: program.filter(line => line.trim() !== ''),
        dataType,
        required
      });
    }
    onClose();
  };



  const handleCancel = () => {
    onClose();
  };
  const handleAddProgramLine = () => {
    if (newProgramLine.trim()) {
      setProgram([...program, newProgramLine.trim()]);
      setNewProgramLine('');
    }
  };

  const handleRemoveProgramLine = (index: number) => {
    setProgram(program.filter((_, i) => i !== index));
  };

  const handleProgramLineChange = (index: number, value: string) => {
    const updated = [...program];
    updated[index] = value;
    setProgram(updated);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-2xl w-full mx-4">
        {/* Title Bar */}
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            AddMapping Properties
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-300 hover:text-gray-800 dark:hover:text-gray-100"
          >
            ✕
          </button>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4 max-h-96 overflow-y-auto">
          {/* Transform Variable */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Transform Variable <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={transform}
              onChange={(e) => setTransform(e.target.value)}
              placeholder="transformVar (naked variable name)"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Use variable name only (e.g. myTransform) - will be used as naked variable
            </p>
          </div>

          {/* Source Field */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Source Field <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={sourceField}
              onChange={(e) => setSourceField(e.target.value)}
              placeholder="incoming_field_name"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Name of the incoming field to map from
            </p>
          </div>

          {/* Target Column */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Target Column <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={targetColumn}
              onChange={(e) => setTargetColumn(e.target.value)}
              placeholder="target_column_name"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Name of the target column to map to
            </p>
          </div>

          {/* Program Array (Optional) */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Transformation Program (Optional)
            </label>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
              String array providing transformation program, one line at a time
            </p>

            {/* Existing Program Lines */}
            {program.length > 0 && (
              <div className="space-y-2 mb-3">
                {program.map((line, index) => (
                  <div key={index} className="flex items-center space-x-2">
                    <span className="text-sm text-gray-500 dark:text-gray-400 w-8">#{index + 1}</span>
                    <input
                      type="text"
                      value={line}
                      onChange={(e) => handleProgramLineChange(index, e.target.value)}
                      className="flex-1 px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="Program line"
                    />
                    <button
                      onClick={() => handleRemoveProgramLine(index)}
                      className="text-red-500 hover:text-red-700 p-1"
                      title="Remove line"
                    >
                      ✕
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* Add New Program Line */}
            <div className="flex items-center space-x-2">
              <input
                type="text"
                value={newProgramLine}
                onChange={(e) => setNewProgramLine(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleAddProgramLine()}
                placeholder="Add transformation line..."
                className="flex-1 px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button
                onClick={handleAddProgramLine}
                disabled={!newProgramLine.trim()}
                className="px-4 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Add
              </button>
            </div>
          </div>

          {/* Data Type */}
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
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Target data type for the mapped field
            </p>
          </div>

          {/* Required Flag */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Required Field
            </label>
            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="required"
                checked={required}
                onChange={(e) => setRequired(e.target.checked)}
                className="w-4 h-4 text-blue-600 bg-gray-100 border-gray-300 rounded focus:ring-blue-500 focus:ring-2"
              />
              <label htmlFor="required" className="text-sm text-gray-700 dark:text-gray-300">
                This field is required
              </label>
            </div>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Whether this mapping is required (true/false)
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
      <div className="absolute top-1/2 left-1/2 transform translate-x-96 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          The AddMapping logicon adds a field mapping to a transform object, defining how data flows from source to target.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. Transform (required, naked variable)</li>
          <li>2. Source field (required, string)</li>
          <li>3. Target column (required, string)</li>
          <li>4. Program (optional, string array)</li>
          <li>5. Data type (required, type)</li>
          <li>6. Required (required, boolean)</li>
        </ul>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          💡 Examples:<br/>
          addMapping(myTransform, "name", "full_name", [], "string", true)<br/>
          addMapping(transform, "age", "user_age", ["parseInt(value)"], "int", false)
        </p>
      </div>
    </div>
  );
};

export default AddMappingNodeProperties;
