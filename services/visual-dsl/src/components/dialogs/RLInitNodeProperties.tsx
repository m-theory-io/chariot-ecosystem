import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface RLInitNodeProperties {
  feat_dim: string;
  alpha: string;
  model_path?: string;
  model_input?: string;
  model_output?: string;
}

interface RLInitNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: RLInitNodeProperties) => void;
  onDelete: () => void;
  initialProperties: RLInitNodeProperties;
}

export const RLInitNodePropertiesDialog: React.FC<RLInitNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [feat_dim, setFeatDim] = React.useState(initialProperties.feat_dim || '12');
  const [alpha, setAlpha] = React.useState(initialProperties.alpha || '0.3');
  const [model_path, setModelPath] = React.useState(initialProperties.model_path || '');
  const [model_input, setModelInput] = React.useState(initialProperties.model_input || '');
  const [model_output, setModelOutput] = React.useState(initialProperties.model_output || '');

  const handleSave = () => {
    onSave({
      feat_dim: feat_dim.trim(),
      alpha: alpha.trim(),
      model_path: model_path.trim() || undefined,
      model_input: model_input.trim() || undefined,
      model_output: model_output.trim() || undefined
    });
    onClose();
  };

  const handleClose = () => {
    onSave({
      feat_dim: feat_dim.trim(),
      alpha: alpha.trim(),
      model_path: model_path.trim() || undefined,
      model_input: model_input.trim() || undefined,
      model_output: model_output.trim() || undefined
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
        {/* Title Bar */}
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            RL Init Properties
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6 space-y-4">
          {/* Feature Dimension */}
          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
              Feature Dimension (required)
            </label>
            <Input
              type="text"
              value={feat_dim}
              onChange={(e) => setFeatDim(e.target.value)}
              placeholder="12"
              className="w-full border-2 border-gray-800 dark:border-gray-200"
            />
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
              Number of features per candidate
            </p>
          </div>

          {/* Alpha */}
          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
              Alpha (required)
            </label>
            <Input
              type="text"
              value={alpha}
              onChange={(e) => setAlpha(e.target.value)}
              placeholder="0.3"
              className="w-full border-2 border-gray-800 dark:border-gray-200"
            />
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
              LinUCB exploration parameter (0.0-1.0)
            </p>
          </div>

          {/* Model Path (Optional) */}
          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
              Model Path (optional)
            </label>
            <Input
              type="text"
              value={model_path}
              onChange={(e) => setModelPath(e.target.value)}
              placeholder="/models/nba.onnx"
              className="w-full border-2 border-gray-800 dark:border-gray-200"
            />
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
              Path to ONNX model file
            </p>
          </div>

          {/* Model Input (Optional) */}
          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
              Model Input Name (optional)
            </label>
            <Input
              type="text"
              value={model_input}
              onChange={(e) => setModelInput(e.target.value)}
              placeholder="input"
              className="w-full border-2 border-gray-800 dark:border-gray-200"
            />
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
              ONNX input tensor name
            </p>
          </div>

          {/* Model Output (Optional) */}
          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
              Model Output Name (optional)
            </label>
            <Input
              type="text"
              value={model_output}
              onChange={(e) => setModelOutput(e.target.value)}
              placeholder="output"
              className="w-full border-2 border-gray-800 dark:border-gray-200"
            />
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
              ONNX output tensor name
            </p>
          </div>

          {/* Action Buttons */}
          <div className="flex gap-3 pt-2">
            <Button
              onClick={handleClose}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Save Properties
            </Button>
            <Button
              onClick={handleDelete}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Delete
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};
