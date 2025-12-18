import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface DivNodeProperties {
  leftOperand: string;
  rightOperand: string;
}

interface DivNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: DivNodeProperties) => void;
  onDelete: () => void;
  initialProperties: DivNodeProperties;
}

export const DivNodePropertiesDialog: React.FC<DivNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [leftOperand, setLeftOperand] = React.useState(initialProperties.leftOperand || 'numerator');
  const [rightOperand, setRightOperand] = React.useState(initialProperties.rightOperand || 'denominator');

  const canSave = leftOperand.trim().length > 0 && rightOperand.trim().length > 0;

  const handleSave = () => {
    if (!canSave) {
      alert('Both operands are required.');
      return;
    }
    onSave({
      leftOperand: leftOperand.trim(),
      rightOperand: rightOperand.trim()
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
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">div() Properties</h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-4">
          <p className="text-sm text-gray-600 dark:text-gray-300">
            Configure numerator and denominator expressions. The <code>div()</code> helper performs runtime-safe division.
          </p>

          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Numerator</label>
            <Input
              type="text"
              value={leftOperand}
              onChange={(e) => setLeftOperand(e.target.value)}
              placeholder="valueA"
              className="w-full"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Denominator</label>
            <Input
              type="text"
              value={rightOperand}
              onChange={(e) => setRightOperand(e.target.value)}
              placeholder="valueB"
              className="w-full"
            />
          </div>

          <p className="text-xs text-gray-500 dark:text-gray-400">
            Ensure the denominator cannot be zero to avoid runtime errors.
          </p>

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
