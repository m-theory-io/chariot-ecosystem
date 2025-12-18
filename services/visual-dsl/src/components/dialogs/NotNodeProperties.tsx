import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface NotNodeProperties {
  operands: string[];
  operand?: string; // legacy single-operand payloads
}

interface NotNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: NotNodeProperties) => void;
  onDelete: () => void;
  initialProperties: NotNodeProperties;
}

export const NotNodePropertiesDialog: React.FC<NotNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [operands, setOperands] = React.useState<string[]>(() => {
    if (Array.isArray(initialProperties.operands) && initialProperties.operands.length > 0) {
      return initialProperties.operands;
    }
    const legacy = (initialProperties.operand ?? '').toString().trim();
    return legacy ? [legacy] : ['flag'];
  });

  const updateOperand = (index: number, value: string) => {
    setOperands((prev) => prev.map((entry, idx) => (idx === index ? value : entry)));
  };

  const addOperand = () => {
    setOperands((prev) => [...prev, '']);
  };

  const removeOperand = (index: number) => {
    setOperands((prev) => prev.filter((_, idx) => idx !== index));
  };

  const sanitizedOperands = operands.map((value) => value.trim()).filter((value) => value.length > 0);
  const canSave = sanitizedOperands.length >= 1;

  const handleSave = () => {
    if (!canSave) {
      alert('Provide at least one boolean expression to negate.');
      return;
    }
    onSave({ operands: sanitizedOperands });
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
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-xl w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">not() Properties</h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-4">
          <p className="text-sm text-gray-600 dark:text-gray-300">
            List each expression to be negated. The <code>not()</code> helper returns <code>false</code> as soon as any operand evaluates to <code>true</code>.
          </p>

          <div className="space-y-3">
            {operands.map((value, index) => (
              <div className="flex items-center gap-2" key={`not-operand-${index}`}>
                <span className="text-xs font-semibold text-gray-500 dark:text-gray-400 w-10">#{index + 1}</span>
                <Input
                  type="text"
                  value={value}
                  onChange={(e) => updateOperand(index, e.target.value)}
                  placeholder={`condition${index + 1}`}
                  className="flex-1"
                />
                {operands.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeOperand(index)}
                    className="text-red-500 hover:text-red-700 px-2"
                    aria-label="Remove operand"
                  >
                    ✕
                  </button>
                )}
              </div>
            ))}
          </div>

          <div className="flex justify-between items-center">
            <Button
              onClick={addOperand}
              className="px-4 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              + Add Operand
            </Button>
            <p className="text-xs text-gray-500 dark:text-gray-400">Minimum of one operand.</p>
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
