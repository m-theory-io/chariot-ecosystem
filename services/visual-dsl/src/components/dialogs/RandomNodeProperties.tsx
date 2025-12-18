import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export type RandomMode = 'unbounded' | 'maxOnly' | 'range';

export interface RandomNodeProperties {
  mode?: RandomMode;
  operands?: string[];
}

interface RandomNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: RandomNodeProperties) => void;
  onDelete: () => void;
  initialProperties: RandomNodeProperties;
}

const inferMode = (props: RandomNodeProperties): RandomMode => {
  const opCount = Array.isArray(props.operands) ? props.operands.length : 0;
  if (opCount >= 2) {
    return 'range';
  }
  if (opCount === 1) {
    return 'maxOnly';
  }
  return props.mode || 'unbounded';
};

export const RandomNodePropertiesDialog: React.FC<RandomNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const initialOperands = Array.isArray(initialProperties.operands) ? initialProperties.operands : [];
  const [mode, setMode] = React.useState<RandomMode>(() => inferMode(initialProperties));
  const [maxOnlyValue, setMaxOnlyValue] = React.useState(() => initialOperands.length === 1 ? initialOperands[0] : 'limit');
  const [rangeMin, setRangeMin] = React.useState(() => initialOperands.length >= 2 ? initialOperands[0] : 'minValue');
  const [rangeMax, setRangeMax] = React.useState(() => initialOperands.length >= 2 ? initialOperands[1] : 'maxValue');

  const canSave =
    mode === 'unbounded' ||
    (mode === 'maxOnly' && maxOnlyValue.trim().length > 0) ||
    (mode === 'range' && rangeMin.trim().length > 0 && rangeMax.trim().length > 0);

  const handleSave = () => {
    let operands: string[] = [];
    if (mode === 'maxOnly') {
      operands = [maxOnlyValue.trim()];
    } else if (mode === 'range') {
      operands = [rangeMin.trim(), rangeMax.trim()];
    }
    onSave({ mode, operands });
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

  const renderModeButton = (value: RandomMode, label: string, description: string) => {
    const isActive = mode === value;
    return (
      <button
        type="button"
        onClick={() => setMode(value)}
        className={`flex-1 px-3 py-2 border ${isActive ? 'bg-gray-900 text-white dark:bg-gray-100 dark:text-gray-900 border-gray-900 dark:border-gray-100' : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200 border-gray-800 dark:border-gray-200'}`}
      >
        <span className="block font-semibold text-sm">{label}</span>
        <span className="block text-xs opacity-80">{description}</span>
      </button>
    );
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-xl w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">random() Properties</h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
            aria-label="Close"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-5">
          <p className="text-sm text-gray-600 dark:text-gray-300">
            Choose how many bounds to provide. <code>random()</code> mirrors the Go runtime helper: no arguments returns a float in [0,1), one argument scales the upper bound, and two arguments generate a value between min and max.
          </p>

          <div className="flex gap-2">
            {renderModeButton('unbounded', 'Unbounded', 'random()')}
            {renderModeButton('maxOnly', 'Max Only', 'random(max)')}
            {renderModeButton('range', 'Range', 'random(min, max)')}
          </div>

          {mode === 'maxOnly' && (
            <div>
              <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Upper Bound</label>
              <Input
                type="text"
                value={maxOnlyValue}
                onChange={(e) => setMaxOnlyValue(e.target.value)}
                placeholder="limit"
                className="w-full"
              />
            </div>
          )}

          {mode === 'range' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Min (inclusive)</label>
                <Input
                  type="text"
                  value={rangeMin}
                  onChange={(e) => setRangeMin(e.target.value)}
                  placeholder="minValue"
                  className="w-full"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Max (exclusive)</label>
                <Input
                  type="text"
                  value={rangeMax}
                  onChange={(e) => setRangeMax(e.target.value)}
                  placeholder="maxValue"
                  className="w-full"
                />
              </div>
            </div>
          )}

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
