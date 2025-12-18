import React, { useState } from 'react';

export interface DoETLNodeProperties {
  jobId: string;
  csvFile: string;
  transformConfig: string;
  targetConfig: string;
  options?: string;
}

interface DoETLNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: DoETLNodeProperties) => void;
  initialProperties?: Partial<DoETLNodeProperties>;
}

const DoETLNodePropertiesDialog: React.FC<DoETLNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {}
}) => {
  const [jobId, setJobId] = useState(initialProperties.jobId || '');
  const [csvFile, setCsvFile] = useState(initialProperties.csvFile || '');
  const [transformConfig, setTransformConfig] = useState(initialProperties.transformConfig || '');
  const [targetConfig, setTargetConfig] = useState(initialProperties.targetConfig || '');
  const [options, setOptions] = useState(initialProperties.options || '');

  if (!isOpen) return null;

  const buildPayload = (): DoETLNodeProperties => ({
    jobId: jobId.trim(),
    csvFile: csvFile.trim(),
    transformConfig: transformConfig.trim(),
    targetConfig: targetConfig.trim(),
    options: options.trim() || undefined
  });

  const validate = () => {
    if (!jobId.trim()) {
      alert('Job ID is required');
      return false;
    }
    if (!csvFile.trim()) {
      alert('CSV file path is required');
      return false;
    }
    if (!transformConfig.trim()) {
      alert('Transform config reference is required');
      return false;
    }
    if (!targetConfig.trim()) {
      alert('Target config reference is required');
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
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-3xl w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            doETL Properties
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-300 hover:text-gray-800 dark:hover:text-gray-100"
          >
            ✕
          </button>
        </div>

        <div className="p-5 space-y-4 max-h-[34rem] overflow-y-auto">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Job ID <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={jobId}
              onChange={(e) => setJobId(e.target.value)}
              placeholder="`etl_${Date.now()}` or jobIdVar"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Literal string or expression passed as the unique ETL job identifier
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              CSV File <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={csvFile}
              onChange={(e) => setCsvFile(e.target.value)}
              placeholder='"data/inventory.csv" or csvPath'
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Path relative to secure data/ directory or a variable reference
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Transform Config <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={transformConfig}
              onChange={(e) => setTransformConfig(e.target.value)}
              placeholder="transformVar"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Accepts a Transform object, TreeNode config, or Map describing mappings
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Target Config <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={targetConfig}
              onChange={(e) => setTargetConfig(e.target.value)}
              placeholder="targetConfigMap"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Typically a map describing SQL/Couchbase connection info
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Options (optional)
            </label>
            <textarea
              value={options}
              onChange={(e) => setOptions(e.target.value)}
              placeholder="map('batchSize', 1000, 'hasHeaders', true) or optionsVar"
              rows={4}
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Optional map for overrides such as delimiter, hasHeaders, clientId, etc.
            </p>
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

      <div className="absolute top-1/2 left-1/2 transform translate-x-96 -translate-y-1/2 max-w-md p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          doETL(jobId, csvFile, transformConfig, targetConfig [, options]) executes the full ETL pipeline.
        </p>
        <p className="text-xs text-gray-600 dark:text-gray-400 mt-2 space-y-1">
          <span className="block">1. Job ID string/expression</span>
          <span className="block">2. CSV filename or path</span>
          <span className="block">3. Transform/TreeNode config</span>
          <span className="block">4. Target database config map</span>
          <span className="block">5. Optional options map (delimiter, hasHeaders, etc.)</span>
        </p>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          Example: doETL('import_users', 'data/users.csv', userTransform, targetSqlConfig, map('batchSize', 500))
        </p>
      </div>
    </div>
  );
};

export default DoETLNodePropertiesDialog;
