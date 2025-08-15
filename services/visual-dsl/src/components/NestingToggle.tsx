import React from 'react';
import { useNesting } from '../contexts/NestingContext';
import { useFlowControl } from '../contexts/FlowControlContext';
import { Button } from './ui/button';

export const NestingToggle: React.FC = () => {
  const { nestingMode, setNestingMode, setSelectedParentId } = useNesting();
  const { selectedNodeId } = useFlowControl();

  const handleToggle = () => {
    if (!nestingMode) {
      // Entering nesting mode
      if (selectedNodeId) {
        setSelectedParentId(selectedNodeId);
        setNestingMode(true);
      } else {
        alert('Please select a parent node first!');
      }
    } else {
      // Exiting nesting mode
      setNestingMode(false);
      setSelectedParentId(null);
    }
  };

  return (
    <Button
      onClick={handleToggle}
      className={nestingMode ? 'bg-yellow-600 hover:bg-yellow-700 dark:bg-yellow-500 dark:hover:bg-yellow-600' : ''}
      title={
        nestingMode 
          ? 'Exit nesting mode' 
          : selectedNodeId 
            ? 'Enter nesting mode for selected node' 
            : 'Select a parent node first, then click to enter nesting mode'
      }
    >
      🪆 {nestingMode ? 'Exit Nesting' : 'Create Nesting'}
    </Button>
  );
};
