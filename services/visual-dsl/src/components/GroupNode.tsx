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
  rounded-3xl 
  bg-transparent
  shadow-none
        min-w-[280px] min-h-[200px]
        pointer-events-none
      "
      style={{
        position: 'absolute',
        zIndex: -1,
      }}
    >
      {/* Minimal visual only container. Title/controls removed per design. */}
      <span className="sr-only">{data.label}</span>
    </div>
  );
};
