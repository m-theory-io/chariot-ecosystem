import React from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { StartNodeProperties } from './dialogs/StartNodeProperties';
import { DeclareNodeProperties } from './dialogs/DeclareNodeProperties';
import { CreateNodeProperties } from './dialogs/CreateNodeProperties';

interface LogiconNodeData {
  label: string;
  icon: string;
  category: string;
  isInNestingGroup?: boolean;
  isNestingParent?: boolean;
  isCurrentNestingTarget?: boolean; // New property for active parent selection
  properties?: StartNodeProperties | DeclareNodeProperties | CreateNodeProperties | Record<string, any>; // Properties for different node types
}

export const LogiconNode: React.FC<NodeProps<LogiconNodeData>> = ({ data, selected }) => {
  const getNodeStyling = () => {
    let baseClasses = "px-3 py-2 shadow-md rounded-lg border-2 min-w-[100px] max-w-[140px]";
    
    // Current nesting target (active parent in nesting mode)
    if (data.isCurrentNestingTarget) {
      baseClasses += " bg-gradient-to-br from-yellow-50 to-yellow-100 dark:from-yellow-900/40 dark:to-yellow-800/40";
      baseClasses += " border-yellow-500 dark:border-yellow-400 shadow-lg shadow-yellow-200 dark:shadow-yellow-800/50";
      baseClasses += " ring-2 ring-yellow-300 dark:ring-yellow-500 ring-offset-2";
    } else if (data.isInNestingGroup) {
      if (data.isNestingParent) {
        // Parent node in nesting group - purple theme
        baseClasses += " bg-gradient-to-br from-purple-50 to-purple-100 dark:from-purple-900/30 dark:to-purple-800/30";
        if (selected) {
          baseClasses += " border-purple-500 dark:border-purple-400 shadow-lg shadow-purple-200 dark:shadow-purple-800/50";
        } else {
          baseClasses += " border-purple-400 dark:border-purple-500";
        }
      } else {
        // Child node in nesting group - lighter purple theme
        baseClasses += " bg-gradient-to-br from-purple-25 to-purple-50 dark:from-purple-950/20 dark:to-purple-900/20";
        if (selected) {
          baseClasses += " border-purple-400 dark:border-purple-300 shadow-lg shadow-purple-100 dark:shadow-purple-700/30";
        } else {
          baseClasses += " border-purple-300 dark:border-purple-600";
        }
      }
    } else {
      // Regular node
      baseClasses += " bg-white dark:bg-gray-800";
      if (selected) {
        baseClasses += " border-blue-500 dark:border-blue-400";
      } else {
        baseClasses += " border-gray-300 dark:border-gray-600";
      }
    }
    
    return baseClasses;
  };

  return (
    <div className={getNodeStyling()}>
      {/* Nesting indicator */}
      {(data.isInNestingGroup || data.isCurrentNestingTarget) && (
        <div className={`absolute -top-2 -right-2 w-5 h-5 rounded-full flex items-center justify-center text-xs font-bold shadow-md ${
          data.isCurrentNestingTarget 
            ? 'bg-yellow-500 dark:bg-yellow-400 text-white animate-pulse' 
            : 'bg-purple-500 dark:bg-purple-400 text-white'
        }`}>
          {data.isCurrentNestingTarget ? '🎯' : data.isNestingParent ? '🪆' : '∈'}
        </div>
      )}
      
      {/* Top Handle */}
      <Handle
        type="target"
        position={Position.Top}
        id="top"
        className="w-2 h-2 !bg-blue-500 border-2 border-white"
      />
      
      {/* Left Handle */}
      <Handle
        type="target"
        position={Position.Left}
        id="left"
        className="w-2 h-2 !bg-blue-500 border-2 border-white"
      />
      
      {/* Right Handle */}
      <Handle
        type="source"
        position={Position.Right}
        id="right"
        className="w-2 h-2 !bg-blue-500 border-2 border-white"
      />
      
      <div className="flex flex-col items-center gap-1">
        <div className="text-lg">{data.icon}</div>
        <div className={`text-xs font-medium text-center ${
          data.isInNestingGroup 
            ? 'text-purple-800 dark:text-purple-200' 
            : 'text-gray-800 dark:text-gray-200'
        }`}>
          {data.label}
        </div>
        
        {/* Nesting label for parent nodes */}
        {data.isNestingParent && (
          <div className="text-xs text-purple-600 dark:text-purple-400 font-semibold">
            (nested)
          </div>
        )}
      </div>
      
      {/* Bottom Handle */}
      <Handle
        type="source"
        position={Position.Bottom}
        id="bottom"
        className="w-2 h-2 !bg-blue-500 border-2 border-white"
      />
    </div>
  );
};
