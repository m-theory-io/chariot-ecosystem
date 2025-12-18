import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface DateAddNodeProperties {
  value?: string;
  interval?: string;
  amount?: string;
}

interface DateAddNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: DateAddNodeProperties) => void;
  onDelete: () => void;
  initialProperties: DateAddNodeProperties;
}

const intervalOptions = [
  { value: 'year', label: 'Years' },
  { value: 'month', label: 'Months' },
  { value: 'day', label: 'Days' },
  { value: 'hour', label: 'Hours' },
  { value: 'minute', label: 'Minutes' },
  { value: 'second', label: 'Seconds' }
];

const coerceText = (value: unknown, fallback: string) => {
  const text = (value ?? '').toString();
  return text.trim().length > 0 ? text : fallback;
};

export const DateAddNodePropertiesDialog: React.FC<DateAddNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [value, setValue] = React.useState(() => coerceText(initialProperties.value, 'now()'));
  const [interval, setInterval] = React.useState(() => coerceText(initialProperties.interval, 'day'));
  const [amount, setAmount] = React.useState(() => coerceText(initialProperties.amount, '1'));

  React.useEffect(() => {
    setValue(coerceText(initialProperties.value, 'now()'));
    setInterval(coerceText(initialProperties.interval, 'day'));
    setAmount(coerceText(initialProperties.amount, '1'));
  }, [initialProperties]);

  if (!isOpen) return null;

  const trimmedValue = value.trim();
  const trimmedInterval = interval.trim();
  const trimmedAmount = amount.trim();
  const canSave = trimmedValue.length > 0 && trimmedInterval.length > 0 && trimmedAmount.length > 0;

  const handleSave = () => {
    if (!canSave) {
      alert('Provide a base date, interval, and amount.');
      return;
    }
    onSave({ value: trimmedValue, interval: trimmedInterval, amount: trimmedAmount });
    onClose();
  };

  const handleCancel = () => {
    onClose();
  };

  const handleDelete = () => {
    onDelete();
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-2xl w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">dateAdd() Properties</h3>
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
            <code>dateAdd(date, interval, value)</code> mirrors the Go helper. Supply a base date expression, choose an interval, and provide a numeric delta. Intervals are case-insensitive and pluralized automatically when needed.
          </p>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Base Date Expression</label>
              <Input
                type="text"
                value={value}
                onChange={(e) => setValue(e.target.value)}
                placeholder="order.shipDate"
                className="w-full"
              />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Interval</label>
                <select
                  value={interval}
                  onChange={(e) => setInterval(e.target.value)}
                  className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400"
                >
                  {intervalOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Runtime accepts singular or plural forms (year/years, etc.).</p>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Amount</label>
                <Input
                  type="text"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  placeholder="1"
                  className="w-full"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Use positive or negative numbers to move forward/backward.</p>
              </div>
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
