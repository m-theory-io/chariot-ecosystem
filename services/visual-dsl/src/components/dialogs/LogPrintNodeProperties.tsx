import React, { useState, useEffect } from 'react';

export interface LogPrintNodeProperties {
  message: string;
  logLevel: string;
  additionalArgs: string[];
}

interface LogPrintNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: LogPrintNodeProperties) => void;
  initialProperties?: Partial<LogPrintNodeProperties>;
}

const LogPrintNodeProperties: React.FC<LogPrintNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  initialProperties = {}
}) => {
  const defaultState: LogPrintNodeProperties = {
    message: '',
    logLevel: 'info',
    additionalArgs: []
  };

  const hydrate = (props: Partial<LogPrintNodeProperties> = {}): LogPrintNodeProperties => {
    const merged: LogPrintNodeProperties = {
      ...defaultState,
      ...props,
      additionalArgs: Array.isArray(props.additionalArgs) ? props.additionalArgs.slice() : []
    };

    merged.message = (merged.message ?? '').toString();
    merged.logLevel = ['debug', 'info', 'warn', 'error'].includes(merged.logLevel) ? merged.logLevel : 'info';
    merged.additionalArgs = merged.additionalArgs.map(arg => (arg ?? '').toString());
    return merged;
  };

  const initialState = hydrate(initialProperties);

  const [message, setMessage] = useState(initialState.message);
  const [logLevel, setLogLevel] = useState(initialState.logLevel);
  const [additionalArgs, setAdditionalArgs] = useState<string[]>(initialState.additionalArgs);
  const [newArg, setNewArg] = useState('');

  const logLevels = ['debug', 'info', 'warn', 'error'];

  useEffect(() => {
    if (isOpen) {
      const hydrated = hydrate(initialProperties);
      setMessage(hydrated.message);
      setLogLevel(hydrated.logLevel);
      setAdditionalArgs(hydrated.additionalArgs);
      setNewArg('');
    }
  }, [isOpen, initialProperties]);

  const handleSave = () => {
    if (!message.trim()) {
      alert('Message is required');
      return;
    }

    onSave({
      message: message.trim(),
      logLevel,
      additionalArgs: additionalArgs.filter(arg => arg.trim() !== '')
    });
    onClose();
  };

  const handleAddArg = () => {
    const trimmed = newArg.trim();
    if (!trimmed) {
      return;
    }
    if (!logLevel.trim()) {
      alert('Specify a log level before adding additional arguments.');
      return;
    }
    setAdditionalArgs([...additionalArgs, trimmed]);
    setNewArg('');
  };

  const handleRemoveArg = (index: number) => {
    setAdditionalArgs(additionalArgs.filter((_, i) => i !== index));
  };

  const handleArgChange = (index: number, value: string) => {
    const updated = [...additionalArgs];
    updated[index] = value;
    setAdditionalArgs(updated);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        {/* Title Bar */}
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            LogPrint Properties
          </h3>
          <button
            onClick={onClose}
            className="text-gray-600 dark:text-gray-300 hover:text-gray-800 dark:hover:text-gray-100"
          >
            ✕
          </button>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4">
          {/* Message Input */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Message <span className="text-red-500">*</span>
            </label>
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Enter message text or variable name"
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={3}
              required
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Use variable names (e.g. msgVar) for variables, or text that will be quoted as strings
            </p>
          </div>

          {/* Log Level Selection */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Log Level
            </label>
            <select
              value={logLevel}
              onChange={(e) => setLogLevel(e.target.value)}
              className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {logLevels.map((level) => (
                <option key={level} value={level}>
                  {level}
                </option>
              ))}
            </select>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Defaults to 'info' if not specified
            </p>
          </div>

          {/* Additional Arguments */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Additional Arguments (Optional)
            </label>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
              Input additional argument and click Add. Additional arguments can only be added if log level is explicitly specified.
            </p>

            {/* Existing Arguments */}
            {additionalArgs.length > 0 && (
              <div className="space-y-2 mb-3">
                {additionalArgs.map((arg, index) => (
                  <div key={index} className="flex items-center space-x-2">
                    <span className="text-sm text-gray-500 dark:text-gray-400 w-8">#{index + 3}</span>
                    <input
                      type="text"
                      value={arg}
                      onChange={(e) => handleArgChange(index, e.target.value)}
                      className="flex-1 px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="Additional argument"
                    />
                    <button
                      onClick={() => handleRemoveArg(index)}
                      className="text-red-500 hover:text-red-700 p-1"
                      title="Remove argument"
                    >
                      ✕
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* Add New Argument */}
            <div className="flex items-center space-x-2">
              <input
                type="text"
                value={newArg}
                onChange={(e) => setNewArg(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleAddArg()}
                placeholder="Input additional argument and click Add..."
                className="flex-1 px-3 py-2 bg-white dark:bg-gray-900 border border-gray-800 dark:border-gray-200 rounded text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button
                onClick={handleAddArg}
                disabled={!newArg.trim() || !logLevel.trim()}
                className="px-4 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Add
              </button>
            </div>
          </div>

          {/* Buttons */}
          <div className="flex gap-3 pt-2">
            <button
              onClick={handleSave}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Save Properties
            </button>
            <button
              onClick={onClose}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
      
      {/* Explanatory text */}
      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          The LogPrint logicon outputs messages to the log with optional severity levels and additional context arguments.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. Message (required, any type converted with %v)</li>
          <li>2. Log level (optional: debug|info|warn|error)</li>
          <li>3-n. Additional arguments (optional)</li>
        </ul>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          💡 Examples:<br/>
          logPrint("Hello World") → logs with info level<br/>
          logPrint("Error occurred", "error") → logs with error level<br/>
          logPrint("User action", "debug", userID, action) → logs with additional context<br/>
          logPrint(count, "warn", "items remaining") → logs number with warning
        </p>
      </div>
    </div>
  );
};

export default LogPrintNodeProperties;
