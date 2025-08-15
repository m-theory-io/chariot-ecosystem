import React from 'react';
import { useFlowControl } from '@/contexts/FlowControlContext';
import { Button } from '@/components/ui/button';

export const DirectionToggle: React.FC = () => {
  const { direction, setDirection } = useFlowControl();

  const directions = [
    { key: 'down' as const, icon: '⬇️', label: 'Vertical Flow', description: 'Bottom → Top connection' },
    { key: 'right' as const, icon: '➡️', label: 'Right Flow', description: 'Right → Left connection' },
    { key: 'up' as const, icon: '⬆️', label: 'Upward Flow', description: 'Top → Bottom connection' }
  ];

  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs font-medium text-gray-600 dark:text-gray-300 uppercase tracking-wide">
        Flow Direction
      </span>
      <div className="flex gap-1">
        {directions.map(({ key, icon, label, description }) => (
          <Button
            key={key}
            onClick={() => setDirection(key)}
            className={`p-2 text-sm ${
              direction === key 
                ? 'bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600' 
                : 'bg-gray-200 hover:bg-gray-300 dark:bg-gray-600 dark:hover:bg-gray-500 text-gray-700 dark:text-gray-200'
            }`}
            title={`${label}: ${description}`}
          >
            {icon}
          </Button>
        ))}
      </div>
    </div>
  );
};
