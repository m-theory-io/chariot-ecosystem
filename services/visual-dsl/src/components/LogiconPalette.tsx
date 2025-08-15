import React, { useState } from 'react';
import { logiconDefinitions, LogiconData } from '@/data/logicons';
import { Button } from '@/components/ui/button';

interface LogiconPaletteProps {
  onAddLogiconFlow: (logicon: LogiconData) => void;
  onAddLogiconRandom: (logicon: LogiconData, withModifier: boolean) => void;
}

export const LogiconPalette: React.FC<LogiconPaletteProps> = ({ onAddLogiconFlow, onAddLogiconRandom }) => {
  // Reorder categories to prioritize structure over procedural details
  const categories = [
    // Structural/Architectural functions first
    'tree', 'node', 'control', 'array', 
    // Data manipulation and construction
    'value', 'json', 'file', 'couchbase', 'sql',
    // Infrastructure and system
    'system', 'host', 'dispatcher', 'etl',
    // Procedural/leaf functions last
    'comparison', 'math', 'string', 'date', 'crypto'
  ] as const;

  // State to track which categories are expanded - start with none expanded
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(
    new Set() // Start with no categories expanded
  );

  const toggleCategory = (category: string) => {
    const newExpanded = new Set(expandedCategories);
    if (newExpanded.has(category)) {
      newExpanded.delete(category);
    } else {
      newExpanded.add(category);
    }
    setExpandedCategories(newExpanded);
  };
  
  const getLogiconsByCategory = (category: string) => 
    logiconDefinitions.filter(logicon => logicon.category === category);

  const getCategoryDisplayName = (category: string) => {
    const names: Record<string, string> = {
      'control': 'Control Flow',
      'array': 'Array',
      'comparison': 'Logic',
      'math': 'Math',
      'string': 'String',
      'value': 'Variables',
      'file': 'File I/O',
      'date': 'Date/Time',
      'crypto': 'Cryptography',
      'system': 'System',
      'couchbase': 'Couchbase',
      'dispatcher': 'Dispatcher',
      'etl': 'ETL',
      'host': 'Host',
      'json': 'JSON',
      'node': 'Node',
      'sql': 'SQL',
      'tree': 'Tree'
    };
    return names[category] || category;
  };

  return (
    <div className="space-y-1 max-h-full overflow-y-auto">
      {categories.map(category => {
        const categoryLogicons = getLogiconsByCategory(category);
        if (categoryLogicons.length === 0) return null;
        
        const isExpanded = expandedCategories.has(category);
        const displayName = getCategoryDisplayName(category);
        
        return (
          <div key={category} className="border-b border-gray-200 dark:border-gray-700 last:border-b-0">
            {/* Category Header - Clickable */}
            <button
              onClick={() => toggleCategory(category)}
              className="w-full flex items-center justify-between py-2 px-1 text-left hover:bg-gray-100 dark:hover:bg-gray-800 rounded transition-colors"
            >
              <h3 className="text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">
                {displayName}
              </h3>
              <div className="flex items-center gap-1">
                <span className="text-xs text-gray-400 dark:text-gray-500">
                  {categoryLogicons.length}
                </span>
                <span className={`text-xs text-gray-400 dark:text-gray-500 transform transition-transform ${
                  isExpanded ? 'rotate-90' : 'rotate-0'
                }`}>
                  ▶
                </span>
              </div>
            </button>
            
            {/* Category Content - Collapsible */}
            {isExpanded && (
              <div className="pb-2 space-y-0.5">
                {categoryLogicons.map((logicon) => (
                  <Button
                    key={logicon.id}
                    className="w-full justify-start text-left text-xs h-8 py-1 px-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border-0"
                    onClick={(e) => {
                      const withModifier = e!.ctrlKey || e!.metaKey || e!.shiftKey;
                      if (withModifier) {
                        onAddLogiconRandom(logicon, true);
                      } else {
                        onAddLogiconFlow(logicon);
                      }
                    }}
                    title={`${logicon.description} | Hold Ctrl/Cmd/Shift for random placement`}
                  >
                    <span className="flex items-center gap-1.5 w-full">
                      <span className="text-sm flex-shrink-0">{logicon.icon}</span>
                      <span className="flex-1 truncate text-xs">{logicon.label}</span>
                    </span>
                  </Button>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};
