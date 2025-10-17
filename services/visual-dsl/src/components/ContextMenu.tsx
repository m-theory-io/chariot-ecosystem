import React from 'react';

interface ContextMenuAction {
  label: string;
  icon?: string;
  onClick: () => void;
  disabled?: boolean;
}

interface ContextMenuProps {
  x: number;
  y: number;
  onDelete: () => void;
  onClose: () => void;
  onProperties?: () => void;
  nodeLabel?: string;
  isGroupRoot?: boolean;
  actions?: ContextMenuAction[];
}

export const ContextMenu: React.FC<ContextMenuProps> = ({ 
  x, 
  y, 
  onDelete, 
  onClose, 
  onProperties,
  nodeLabel,
  isGroupRoot,
  actions = []
}) => {
  React.useEffect(() => {
    const handleClickOutside = () => onClose();
    document.addEventListener('click', handleClickOutside);
    return () => document.removeEventListener('click', handleClickOutside);
  }, [onClose]);

  return (
    <div 
      className="fixed z-50 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg shadow-lg py-1 min-w-[160px]"
      style={{ 
        left: x, 
        top: y,
        transform: 'translate(0, 0)'
      }}
      onClick={(e) => e.stopPropagation()}
    >
      <div className="px-3 py-1 text-xs font-medium text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-600">
        {nodeLabel || 'Node'}
        {isGroupRoot && (
          <span className="ml-1 text-purple-500 dark:text-purple-400">🪆</span>
        )}
      </div>
      
      {actions.map((action, idx) => (
        <button
          key={`${action.label}-${idx}`}
          onClick={(e) => {
            e.stopPropagation();
            if (!action.disabled) {
              action.onClick();
            }
          }}
          className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2 ${
            action.disabled
              ? 'text-gray-400 dark:text-gray-500 cursor-not-allowed'
              : 'text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700'
          }`}
          disabled={action.disabled}
        >
          <span>{action.icon || '➕'}</span>
          <span>{action.label}</span>
        </button>
      ))}

      {onProperties && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            onProperties();
          }}
          className="w-full text-left px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-2"
        >
          <span>⚙️</span>
          <span>Properties...</span>
        </button>
      )}
      
      <button
        onClick={(e) => {
          e.stopPropagation();
          onDelete();
        }}
        className="w-full text-left px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 flex items-center gap-2"
      >
        <span>🗑️</span>
        <span>
          Delete {isGroupRoot ? 'Group' : 'Node'}
          {isGroupRoot && <span className="text-xs text-gray-500 ml-1">(and children)</span>}
        </span>
      </button>
    </div>
  );
};
