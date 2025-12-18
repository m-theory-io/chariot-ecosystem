import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export type DateNodeMode = 'parse' | 'components';

export interface DateNodeProperties {
  mode?: DateNodeMode;
  value?: string;
  year?: string;
  month?: string;
  day?: string;
}

interface DateNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: DateNodeProperties) => void;
  onDelete: () => void;
  initialProperties: DateNodeProperties;
}

const inferMode = (props: DateNodeProperties): DateNodeMode => {
  if (props.mode === 'components') {
    return 'components';
  }
  const yearFilled = (props.year ?? '').toString().trim().length > 0;
  const monthFilled = (props.month ?? '').toString().trim().length > 0;
  const dayFilled = (props.day ?? '').toString().trim().length > 0;
  if (yearFilled && monthFilled && dayFilled) {
    return 'components';
  }
  return 'parse';
};

const coerceText = (value: unknown, fallback: string) => {
  const text = (value ?? '').toString();
  return text.trim().length > 0 ? text : fallback;
};

export const DateNodePropertiesDialog: React.FC<DateNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [mode, setMode] = React.useState<DateNodeMode>(() => inferMode(initialProperties));
  const [value, setValue] = React.useState(() => coerceText(initialProperties.value, '2024-01-01T00:00:00Z'));
  const [year, setYear] = React.useState(() => coerceText(initialProperties.year, '2024'));
  const [month, setMonth] = React.useState(() => coerceText(initialProperties.month, '1'));
  const [day, setDay] = React.useState(() => coerceText(initialProperties.day, '1'));

  React.useEffect(() => {
    setMode(inferMode(initialProperties));
    setValue(coerceText(initialProperties.value, '2024-01-01T00:00:00Z'));
    setYear(coerceText(initialProperties.year, '2024'));
    setMonth(coerceText(initialProperties.month, '1'));
    setDay(coerceText(initialProperties.day, '1'));
  }, [initialProperties]);

  if (!isOpen) return null;

  const trimmedValue = value.trim();
  const trimmedYear = year.trim();
  const trimmedMonth = month.trim();
  const trimmedDay = day.trim();

  const canSave =
    (mode === 'parse' && trimmedValue.length > 0) ||
    (mode === 'components' && trimmedYear.length > 0 && trimmedMonth.length > 0 && trimmedDay.length > 0);

  const handleSave = () => {
    if (!canSave) {
      alert('Provide either a date expression to parse or numeric year/month/day components.');
      return;
    }
    if (mode === 'parse') {
      onSave({ mode, value: trimmedValue });
    } else {
      onSave({ mode, year: trimmedYear, month: trimmedMonth, day: trimmedDay });
    }
    onClose();
  };

  const handleCancel = () => {
    onClose();
  };

  const handleDelete = () => {
    onDelete();
    onClose();
  };

  const renderModeButton = (valueKey: DateNodeMode, label: string, description: string) => {
    const isActive = mode === valueKey;
    return (
      <button
        type="button"
        onClick={() => setMode(valueKey)}
        className={`flex-1 px-3 py-2 border ${isActive ? 'bg-gray-900 text-white dark:bg-gray-100 dark:text-gray-900 border-gray-900 dark:border-gray-100' : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200 border-gray-800 dark:border-gray-200'}`}
      >
        <span className="block font-semibold text-sm">{label}</span>
        <span className="block text-xs opacity-80">{description}</span>
      </button>
    );
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-2xl w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">date() Properties</h3>
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
            The Go runtime accepts either a single timestamp string or three numeric components. Pick a mode and provide the matching fields; the generated DSL will call <code>date(...)</code> with the same argument pattern.
          </p>

          <div className="flex gap-2">
            {renderModeButton('parse', 'Parse String', 'date(source)')}
            {renderModeButton('components', 'Year/Month/Day', 'date(year, month, day)')}
          </div>

          {mode === 'parse' && (
            <div>
              <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Date Expression</label>
              <Input
                type="text"
                value={value}
                onChange={(e) => setValue(e.target.value)}
                placeholder="order.createdAt"
                className="w-full"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Provide a literal or expression that resolves to a string (ISO timestamps, RFC3339, or friendly formats supported by the runtime parser).</p>
            </div>
          )}

          {mode === 'components' && (
            <div>
              <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">Components</label>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Input
                  type="text"
                  value={year}
                  onChange={(e) => setYear(e.target.value)}
                  placeholder="2024"
                  className="w-full"
                />
                <Input
                  type="text"
                  value={month}
                  onChange={(e) => setMonth(e.target.value)}
                  placeholder="1"
                  className="w-full"
                />
                <Input
                  type="text"
                  value={day}
                  onChange={(e) => setDay(e.target.value)}
                  placeholder="1"
                  className="w-full"
                />
              </div>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Each field can be a literal or expression that evaluates to a number.</p>
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
