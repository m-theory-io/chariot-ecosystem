import React, { useState } from 'react';

export interface ETLStatusNodeProperties {
  jobId: string;
}

interface ETLStatusNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: ETLStatusNodeProperties) => void;
  initialProperties?: Partial<ETLStatusNodeProperties>;
}

const ETLStatusNodePropertiesDialog: React.FC<ETLStatusNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {}
}) => {
  const [jobId, setJobId] = useState(initialProperties.jobId || '');

  if (!isOpen) return null;

  const buildPayload = (): ETLStatusNodeProperties => ({
    jobId: jobId.trim()
  });

  const validate = () => {
    if (!jobId.trim()) {
      alert('Job ID is required');
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
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-lg w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            etlStatus Properties
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-300 hover:text-gray-800 dark:hover:text-gray-100"
          >
            ✕
          </button>
        </div>

        <div className="p-5 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Job ID <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={jobId}
              onChange={(e) => setJobId(e.target.value)}
              placeholder="'import_users' or jobIdVar"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Matches the jobId used when calling doETL
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

      <div className="absolute top-1/2 left-1/2 transform translate-x-80 -translate-y-1/2 max-w-xs p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          etlStatus(jobId) fetches the latest ETL log or Couchbase document for the provided job identifier.
        </p>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          Example: etlStatus('import_users')
        </p>
      </div>
    </div>
  );
};

export default ETLStatusNodePropertiesDialog;
