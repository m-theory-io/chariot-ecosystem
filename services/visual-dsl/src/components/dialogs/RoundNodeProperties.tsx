import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface RoundNodeProperties {
  value: string;
  decimalPlaces?: string;
}

interface RoundNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: RoundNodeProperties) => void;
  onDelete: () => void;
  initialProperties: RoundNodeProperties;
}

export const RoundNodePropertiesDialog: React.FC<RoundNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [valueExpression, setValueExpression] = React.useState(initialProperties.value || 'value');
  const [decimalPlaces, setDecimalPlaces] = React.useState(initialProperties.decimalPlaces ?? '');

  const canSave = valueExpression.trim().length > 0;

  const handleSave = () => {
    if (!canSave) {
      alert('Provide a value to round.');
      return;
    }
    onSave({
      value: valueExpression.trim(),
      decimalPlaces: decimalPlaces.trim()
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

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">round() Properties</h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
            aria-label="Close"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-4">
          <p className="text-sm text-gray-600 dark:text-gray-300">
            Specify the source expression and optionally provide decimal places. Leave the <em>Decimal Places</em> field blank to call <code>round()</code> with a single argument.
          </p>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Value</label>
              <Input
                type="text"
                value={valueExpression}
                onChange={(e) => setValueExpression(e.target.value)}
                placeholder="value"
                className="w-full"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Decimal Places (optional)</label>
              <Input
                type="text"
                value={decimalPlaces}
                onChange={(e) => setDecimalPlaces(e.target.value)}
                placeholder="2"
                className="w-full"
              />
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
    </div>
  );
};
