import React, { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface CBQueryNodeProperties {
  query?: string;
  bucket?: string;
  scope?: string;
  collection?: string;
  parameters?: string;
  resultVariable?: string;
  description?: string;
  name?: string;
}

interface CBQueryNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: CBQueryNodeProperties) => void;
  initialProperties?: CBQueryNodeProperties;
}

export const CBQueryNodePropertiesDialog: React.FC<CBQueryNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {}
}) => {
  const [properties, setProperties] = useState<CBQueryNodeProperties>({
    query: '',
    bucket: '',
    scope: '_default',
    collection: '_default',
    parameters: '',
    resultVariable: 'queryResult',
    description: '',
    name: '',
    ...initialProperties
  });

  useEffect(() => {
    if (isOpen) {
      setProperties({
        query: '',
        bucket: '',
        scope: '_default',
        collection: '_default',
        parameters: '',
        resultVariable: 'queryResult',
        description: '',
        name: '',
        ...initialProperties
      });
    }
  }, [isOpen, initialProperties]);

  const handleSave = () => {
    onSave(properties);
    onClose();
  };

  const handleCancel = () => {
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4">
        <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
          Couchbase Query Properties
        </h3>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Name (optional)
            </label>
            <Input
              type="text"
              value={properties.name || ''}
              onChange={(e) => setProperties({ ...properties, name: e.target.value })}
              placeholder="Name for this query"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Bucket Name
            </label>
            <Input
              type="text"
              value={properties.bucket || ''}
              onChange={(e) => setProperties({ ...properties, bucket: e.target.value })}
              placeholder="e.g., my-bucket, users, products"
            />
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Scope
              </label>
              <Input
                type="text"
                value={properties.scope || ''}
                onChange={(e) => setProperties({ ...properties, scope: e.target.value })}
                placeholder="_default"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Collection
              </label>
              <Input
                type="text"
                value={properties.collection || ''}
                onChange={(e) => setProperties({ ...properties, collection: e.target.value })}
                placeholder="_default"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              N1QL Query
            </label>
            <textarea
              value={properties.query || ''}
              onChange={(e) => setProperties({ ...properties, query: e.target.value })}
              placeholder="SELECT * FROM `bucket` WHERE type = $1"
              rows={4}
              className="w-full p-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 font-mono text-sm"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Use $1, $2, etc. for parameterized queries
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Parameters (optional)
            </label>
            <Input
              type="text"
              value={properties.parameters || ''}
              onChange={(e) => setProperties({ ...properties, parameters: e.target.value })}
              placeholder='["value1", "value2"] or {param1: "value1"}'
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              JSON array or object of parameters
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Result Variable
            </label>
            <Input
              type="text"
              value={properties.resultVariable || ''}
              onChange={(e) => setProperties({ ...properties, resultVariable: e.target.value })}
              placeholder="queryResult"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Variable name to store query results
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Description (optional)
            </label>
            <Input
              type="text"
              value={properties.description || ''}
              onChange={(e) => setProperties({ ...properties, description: e.target.value })}
              placeholder="Describe what this query does"
            />
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-6">
          <Button
            onClick={handleCancel}
            className="px-4 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 text-white"
          >
            Save
          </Button>
        </div>
      </div>
    </div>
  );
};

export default CBQueryNodeProperties;
