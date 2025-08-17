import React, { createContext, useContext, useState } from 'react';

type FlowDirection = 'down' | 'right' | 'up';

interface FlowControlContextType {
  direction: FlowDirection;
  setDirection: (direction: FlowDirection) => void;
  selectedNodeId: string | null;
  setSelectedNodeId: (nodeId: string | null) => void;
}

const FlowControlContext = createContext<FlowControlContextType | undefined>(undefined);

export const useFlowControl = () => {
  const context = useContext(FlowControlContext);
  if (context === undefined) {
    throw new Error('useFlowControl must be used within a FlowControlProvider');
  }
  return context;
};

export const FlowControlProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [direction, setDirection] = useState<FlowDirection>('down');
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  return (
    <FlowControlContext.Provider value={{ 
      direction, 
      setDirection, 
      selectedNodeId, 
      setSelectedNodeId 
    }}>
      {children}
    </FlowControlContext.Provider>
  );
};
