import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface RLExploreNodeProperties {
  scores: string;
  candidates: string;
  epsilon: string;
}

interface RLExploreNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: RLExploreNodeProperties) => void;
  onDelete: () => void;
  initialProperties: RLExploreNodeProperties;
}

export const RLExploreNodePropertiesDialog: React.FC<RLExploreNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [scores, setScores] = React.useState(initialProperties.scores || 'scores');
  const [candidates, setCandidates] = React.useState(initialProperties.candidates || 'candidates');
  const [epsilon, setEpsilon] = React.useState(initialProperties.epsilon || '0.1');

  const handleSave = () => {
    onSave({
      scores: scores.trim(),
      candidates: candidates.trim(),
      epsilon: epsilon.trim()
    });
    onClose();
  };

  const handleClose = () => {
    onSave({
      scores: scores.trim(),
      candidates: candidates.trim(),
      epsilon: epsilon.trim()
    });
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
            RL Explore Properties
          </h3>
          <button
            onClick={handleClose}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
              Scores Array
            </label>
            <Input
              type="text"
              value={scores}
              onChange={(e) => setScores(e.target.value)}
              placeholder="scores"
              className="w-full border-2 border-gray-800 dark:border-gray-200"
            />
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
              Variable containing array of scores from rlScore
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
              Candidates Array
            </label>
            <Input
              type="text"
              value={candidates}
              onChange={(e) => setCandidates(e.target.value)}
              placeholder="candidates"
              className="w-full border-2 border-gray-800 dark:border-gray-200"
            />
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
              Variable containing array of candidate objects
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
              Epsilon
            </label>
            <Input
              type="text"
              value={epsilon}
              onChange={(e) => setEpsilon(e.target.value)}
              placeholder="0.1"
              className="w-full border-2 border-gray-800 dark:border-gray-200"
            />
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
              Exploration rate (0.0-1.0): 0.0=exploit only, 1.0=explore only
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
