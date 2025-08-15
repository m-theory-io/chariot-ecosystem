import React from 'react';
import { NodeProps } from 'reactflow';

interface GroupNodeData {
  label: string;
}

export const GroupNode: React.FC<NodeProps<GroupNodeData>> = ({ data, selected }) => {
  console.log('GroupNode rendering:', { data, selected });
  
  return (
    <div 
      className="
        p-6 rounded-xl 
        border-4 border-dashed border-purple-500 dark:border-purple-400
        bg-purple-100/50 dark:bg-purple-900/30
        shadow-xl shadow-purple-200 dark:shadow-purple-800/50
        min-w-[280px] min-h-[200px]
        backdrop-blur-sm
        pointer-events-none
      "
      style={{
        position: 'absolute',
        zIndex: -1,
      }}
    >
      <div className="flex items-center gap-2 mb-3">
        <span className="text-3xl">🪆</span>
        <span className="text-lg font-bold text-purple-700 dark:text-purple-300 bg-white dark:bg-gray-800 px-3 py-1 rounded-md shadow-sm">
          {data.label}
        </span>
      </div>
      
      <div className="text-sm font-medium text-purple-600 dark:text-purple-400 mb-2">
        Nested Function Group
      </div>
      
      {/* Container area for nested nodes */}
      <div className="mt-4 min-h-[120px] rounded-lg border-2 border-purple-300 dark:border-purple-600 bg-white/40 dark:bg-gray-900/20 backdrop-blur-sm">
        <div className="p-3 text-center text-sm text-purple-600 dark:text-purple-400 opacity-75">
          Contains nested nodes
        </div>
      </div>
    </div>
  );
};
