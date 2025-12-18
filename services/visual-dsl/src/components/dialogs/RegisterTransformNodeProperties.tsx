import React, { useMemo, useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

const dataTypes = ['string', 'int', 'float', 'bool', 'date', 'datetime', 'json'];
const categoryHints = ['validation', 'formatting', 'conversion', 'enrichment', 'custom'];

export interface RegisterTransformNodeProperties {
  transformName: string;
  description?: string;
  dataType?: string;
  category?: string;
  program: string[];
}

interface RegisterTransformNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: RegisterTransformNodeProperties) => void;
  onDelete: () => void;
  initialProperties: RegisterTransformNodeProperties;
  availableTransformNames?: string[];
  transformFetchError?: string;
}

export const RegisterTransformNodePropertiesDialog: React.FC<RegisterTransformNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties,
  availableTransformNames = [],
  transformFetchError
}) => {
  const [transformName, setTransformName] = useState(initialProperties.transformName || '');
  const [description, setDescription] = useState(initialProperties.description || '');
  const [dataType, setDataType] = useState(initialProperties.dataType || 'string');
  const [category, setCategory] = useState(initialProperties.category || '');
  const [program, setProgram] = useState<string[]>(Array.isArray(initialProperties.program) ? initialProperties.program : []);
  const [newProgramLine, setNewProgramLine] = useState('');

  const normalizedName = transformName.trim();
  const canSave = normalizedName.length > 0 && program.some((line) => line.trim().length > 0);

  const duplicateName = useMemo(() => {
    if (!normalizedName) {
      return false;
    }
    const lower = normalizedName.toLowerCase();
    return availableTransformNames.some((candidate) => candidate.trim().toLowerCase() === lower);
  }, [availableTransformNames, normalizedName]);

  const handleSave = () => {
    const finalName = normalizedName;
    const sanitizedProgram = program
      .map((line) => line.trim())
      .filter((line) => line.length > 0);

    if (!finalName) {
      alert('Transform name is required.');
      return;
    }

    if (sanitizedProgram.length === 0) {
      alert('Add at least one program line.');
      return;
    }

    onSave({
      transformName: finalName,
      description: description.trim(),
      dataType,
      category: category.trim(),
      program: sanitizedProgram
    });
    onClose();
  };

  const handleCancel = () => {
    onClose();
  };

  const handleDelete = () => {
    onDelete();
    onClose();
  };

  const handleProgramLineChange = (index: number, value: string) => {
    setProgram((prev) => prev.map((line, idx) => (idx === index ? value : line)));
  };

  const handleRemoveProgramLine = (index: number) => {
    setProgram((prev) => prev.filter((_, idx) => idx !== index));
  };

  const handleAddProgramLine = () => {
    const nextLine = newProgramLine.trim();
    if (!nextLine) {
      return;
    }
    setProgram((prev) => [...prev, nextLine]);
    setNewProgramLine('');
  };

  if (!isOpen) return null;

  const showNameWarning = duplicateName && normalizedName.length > 0;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-3xl w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Register Transform
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-6 max-h-[34rem] overflow-y-auto">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Transform Name <span className="text-red-500">*</span>
            </label>
            <Input
              type="text"
              value={transformName}
              onChange={(e) => setTransformName(e.target.value)}
              placeholder="zip5, ssn, customFormatter"
              className="w-full"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Use bare variable names for symbols or plain strings for literal names. The code generator quotes values automatically when needed.
            </p>
            {showNameWarning && (
              <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">
                A transform with this name already exists. Registering again will override it at runtime.
              </p>
            )}
            {transformFetchError && (
              <p className="text-xs text-red-600 dark:text-red-400 mt-1">
                {transformFetchError}
              </p>
            )}
          </div>

          <div className="grid md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Description
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Explains what the transform does"
                className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                rows={4}
              />
            </div>
            <div className="space-y-4">
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
                    <option value={type} key={type}>
                      {type}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Category
                </label>
                <Input
                  type="text"
                  value={category}
                  onChange={(e) => setCategory(e.target.value)}
                  placeholder="validation, formatting, ..."
                  className="w-full"
                  list="register-transform-category"
                />
                <datalist id="register-transform-category">
                  {categoryHints.map((hint) => (
                    <option value={hint} key={hint} />
                  ))}
                </datalist>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Optional grouping label for dashboards and docs.
                </p>
              </div>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Program Lines <span className="text-red-500">*</span>
            </label>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
              Provide one Chariot statement per line. Use <code>sourceValue</code> (or your own variables) to read the incoming value and return the transformed value from the last expression.
            </p>

            {program.length > 0 && (
              <div className="space-y-2 mb-3">
                {program.map((line, index) => (
                  <div className="flex items-center space-x-2" key={`${index}-${line}`}>
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

            <div className="flex items-center space-x-2">
              <input
                type="text"
                value={newProgramLine}
                onChange={(e) => setNewProgramLine(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    handleAddProgramLine();
                  }
                }}
                placeholder="Add new line, press Enter to commit"
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

      <div className="absolute top-1/2 left-1/2 transform translate-x-[30rem] -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          <strong>Signature:</strong> registerTransform(name, config)
        </p>
        <p className="text-xs text-gray-600 dark:text-gray-400 mt-2">
          The <code>config</code> map typically includes <code>description</code>, <code>dataType</code>, <code>category</code>, and a
          <code>program</code> array. The program can call helpers, emit errors, or mutate temporary variables before returning.
        </p>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          Example:<br />registerTransform('zip5', map('description', 'Formats US ZIP', 'dataType', 'string', 'category', 'validation', 'program', array("setq(clean, regexReplace(sourceValue, '[^0-9]', ''))", "if(equal(length(clean), 5), clean, error('Invalid ZIP')))")))
        </p>
      </div>
    </div>
  );
};
