export interface LogiconData {
  id: string;
  label: string;
  icon: string;
  description: string;
  category: 'control' | 'array' | 'comparison' | 'couchbase' | 'date' | 'dispatcher' | 'etl' | 'file' | 'crypto' | 'host' | 'json' | 'math' | 'ml' | 'bdi' | 'node' | 'csv' | 'sql' | 'string' | 'system' | 'tree' | 'value';
}

export const logiconDefinitions: LogiconData[] = [
  // Control Flow
  {
    id: 'start',
    label: 'Start',
    icon: '🚀',
    description: 'Start of program',
    category: 'control'
  },
  {
    id: 'if',
    label: 'If',
    icon: '🔀',
    description: 'Conditional branching statement',
    category: 'control'
  },
  {
    id: 'while',
    label: 'While',
    icon: '⭕',
    description: 'While loop construct',
    category: 'control'
  },
  {
    id: 'func',
    label: 'func',
    icon: '⚙️',
    description: 'Inline Chariot function value, such as func(){ equal(1, 1) }',
    category: 'control'
  },
  {
    id: 'switch',
    label: 'Switch',
    icon: '🔄',
    description: 'Switch statement',
    category: 'control'
  },
  {
    id: 'case',
    label: 'Case',
    icon: '📋',
    description: 'Case in switch statement',
    category: 'control'
  },
  {
    id: 'default',
    label: 'Default',
    icon: '🎯',
    description: 'Default case in switch',
    category: 'control'
  },
  {
    id: 'break',
    label: 'Break',
    icon: '🛑',
    description: 'Break from loop',
    category: 'control'
  },
  {
    id: 'continue',
    label: 'Continue',
    icon: '⏭️',
    description: 'Continue loop iteration',
    category: 'control'
  },

  // BDI / Signal Functions
  {
    id: 'plan',
    label: 'plan',
    icon: '🧭',
    description: 'Create a BDI plan from name, parameters, trigger, guard, steps, and optional drop condition',
    category: 'bdi'
  },
  {
    id: 'belief',
    label: 'belief',
    icon: '💭',
    description: 'Read a belief value from an agent',
    category: 'bdi'
  },
  {
    id: 'agentBelief',
    label: 'agentBelief',
    icon: '📌',
    description: 'Set a belief on a named agent and publish activation',
    category: 'bdi'
  },
  {
    id: 'agentStartNamed',
    label: 'agentStartNamed',
    icon: '▶️',
    description: 'Start a named BDI agent with max concurrency, polling interval, and lifecycle',
    category: 'bdi'
  },
  {
    id: 'agentStopNamed',
    label: 'agentStopNamed',
    icon: '⏹️',
    description: 'Stop a named BDI agent',
    category: 'bdi'
  },
  {
    id: 'agentList',
    label: 'agentList',
    icon: '📋',
    description: 'List named BDI agents',
    category: 'bdi'
  },
  {
    id: 'agentPublish',
    label: 'agentPublish',
    icon: '📣',
    description: 'Publish activation for a named BDI agent',
    category: 'bdi'
  },
  {
    id: 'runPlanOnce',
    label: 'runPlanOnce',
    icon: '🧪',
    description: 'Run a plan once outside a long-running agent',
    category: 'bdi'
  },
  {
    id: 'setStepResult',
    label: 'setStepResult',
    icon: '✅',
    description: 'Set structured output for the current BDI step',
    category: 'bdi'
  },
  {
    id: 'setPlanResult',
    label: 'setPlanResult',
    icon: '🏁',
    description: 'Set structured output for the current BDI plan',
    category: 'bdi'
  },
  {
    id: 'signalRegister',
    label: 'signalRegister',
    icon: '📡',
    description: 'Register a signal source such as static, random, sysfs, or httpJson',
    category: 'bdi'
  },
  {
    id: 'signalRead',
    label: 'signalRead',
    icon: '📥',
    description: 'Read a registered signal source once',
    category: 'bdi'
  },
  {
    id: 'signalList',
    label: 'signalList',
    icon: '📋',
    description: 'List registered signal sources',
    category: 'bdi'
  },
  {
    id: 'signalStartBeliefFeed',
    label: 'signalStartBeliefFeed',
    icon: '🔁',
    description: 'Start a signal feed that updates an agent belief and activates the agent on change',
    category: 'bdi'
  },
  {
    id: 'signalStopBeliefFeed',
    label: 'signalStopBeliefFeed',
    icon: '⏸️',
    description: 'Stop a running signal belief feed',
    category: 'bdi'
  },
  {
    id: 'signalFeedList',
    label: 'signalFeedList',
    icon: '📊',
    description: 'List running signal belief feeds',
    category: 'bdi'
  },

  // Array Functions
  {
    id: 'addTo',
    label: 'Add To',
    icon: '➕',
    description: 'Add element to array',
    category: 'array'
  },
  {
    id: 'array',
    label: 'Array',
    icon: '📊',
    description: 'Create or manipulate array',
    category: 'array'
  },
  {
    id: 'lastIndex',
    label: 'Last Index',
    icon: '🔚',
    description: 'Get last index of array',
    category: 'array'
  },
  {
    id: 'length',
    label: 'Length',
    icon: '📏',
    description: 'Get array length',
    category: 'array'
  },
  {
    id: 'removeAt',
    label: 'Remove At',
    icon: '❌',
    description: 'Remove element at index',
    category: 'array'
  },
  {
    id: 'range',
    label: 'Range',
    icon: '🔢',
    description: 'Generate numeric range array',
    category: 'array'
  },
  {
    id: 'reverse',
    label: 'Reverse',
    icon: '🔄',
    description: 'Reverse array order',
    category: 'array'
  },
  {
    id: 'setAt',
    label: 'Set At',
    icon: '📝',
    description: 'Set element at index',
    category: 'array'
  },
  {
    id: 'slice',
    label: 'Slice',
    icon: '✂️',
    description: 'Extract array slice',
    category: 'array'
  },

  // Comparison Functions
  {
    id: 'and',
    label: 'and',
    icon: '🤝',
    description: 'Logical AND operation',
    category: 'comparison'
  },
  {
    id: 'or',
    label: 'or',
    icon: '🔗',
    description: 'Logical OR operation',
    category: 'comparison'
  },
  {
    id: 'not',
    label: 'not',
    icon: '🚫',
    description: 'Logical NOT operation',
    category: 'comparison'
  },
  {
    id: 'equal',
    label: 'equal',
    icon: '⚖️',
    description: 'Equality comparison',
    category: 'comparison'
  },
  {
    id: 'unequal',
    label: 'unequal',
    icon: '⚔️',
    description: 'Inequality comparison',
    category: 'comparison'
  },
  {
    id: 'bigger',
    label: 'bigger',
    icon: '▶️',
    description: 'Greater than comparison',
    category: 'comparison'
  },
  {
    id: 'biggerEq',
    label: 'biggerEq',
    icon: '⏩',
    description: 'Greater than or equal comparison',
    category: 'comparison'
  },
  {
    id: 'smaller',
    label: 'smaller',
    icon: '◀️',
    description: 'Less than comparison',
    category: 'comparison'
  },
  {
    id: 'smallerEq',
    label: 'smallerEq',
    icon: '⏪',
    description: 'Less than or equal comparison',
    category: 'comparison'
  },
  {
    id: 'iif',
    label: 'iif',
    icon: '❓',
    description: 'Immediate if statement',
    category: 'comparison'
  },

  // Couchbase Functions
  {
    id: 'cbConnect',
    label: 'CB Connect',
    icon: '🔌',
    description: 'Connect to Couchbase',
    category: 'couchbase'
  },
  {
    id: 'cbQuery',
    label: 'CB Query',
    icon: '🔍',
    description: 'Query Couchbase database',
    category: 'couchbase'
  },
  {
    id: 'cbInsert',
    label: 'CB Insert',
    icon: '💾',
    description: 'Insert into Couchbase',
    category: 'couchbase'
  },
  {
    id: 'cbGet',
    label: 'CB Get',
    icon: '📥',
    description: 'Get from Couchbase',
    category: 'couchbase'
  },
  {
    id: 'cbRemove',
    label: 'CB Remove',
    icon: '🗑️',
    description: 'Remove from Couchbase',
    category: 'couchbase'
  },

  // Date Functions
  {
    id: 'date',
    label: 'Date',
    icon: '📅',
    description: 'Date manipulation',
    category: 'date'
  },
  {
    id: 'now',
    label: 'Now',
    icon: '⏰',
    description: 'Current date/time',
    category: 'date'
  },
  {
    id: 'today',
    label: 'Today',
    icon: '📆',
    description: 'Current date',
    category: 'date'
  },
  {
    id: 'dateAdd',
    label: 'Date Add',
    icon: '➕',
    description: 'Add to date',
    category: 'date'
  },
  {
    id: 'formatDate',
    label: 'Format Date',
    icon: '🎨',
    description: 'Format date string',
    category: 'date'
  },

  // Dispatcher Functions
  {
    id: 'apply',
    label: 'Apply',
    icon: '🎯',
    description: 'Apply function to object',
    category: 'dispatcher'
  },
  {
    id: 'clone',
    label: 'Clone',
    icon: '👥',
    description: 'Clone an object',
    category: 'dispatcher'
  },
  {
    id: 'contains',
    label: 'Contains',
    icon: '🔍',
    description: 'Check if contains element',
    category: 'dispatcher'
  },
  {
    id: 'getAllMeta',
    label: 'Get All Meta',
    icon: '📊',
    description: 'Get all metadata',
    category: 'dispatcher'
  },
  {
    id: 'getAt',
    label: 'Get At',
    icon: '📍',
    description: 'Get element at position',
    category: 'dispatcher'
  },
  {
    id: 'getAttributes',
    label: 'Get Attributes',
    icon: '🏷️',
    description: 'Get object attributes',
    category: 'dispatcher'
  },
  {
    id: 'getMeta',
    label: 'Get Meta',
    icon: '📋',
    description: 'Get metadata',
    category: 'dispatcher'
  },
  {
    id: 'getProp',
    label: 'Get Property',
    icon: '🔑',
    description: 'Get object property',
    category: 'dispatcher'
  },
  {
    id: 'indexOf',
    label: 'Index Of',
    icon: '🔢',
    description: 'Find index of element',
    category: 'dispatcher'
  },
  {
    id: 'setMeta',
    label: 'Set Meta',
    icon: '📝',
    description: 'Set metadata',
    category: 'dispatcher'
  },
  {
    id: 'setProp',
    label: 'Set Property',
    icon: '🔧',
    description: 'Set object property',
    category: 'dispatcher'
  },

  // ETL Functions
  {
    id: 'addMapping',
    label: 'Add Mapping',
    icon: '🗺️',
    description: 'Add data mapping',
    category: 'etl'
  },
  {
    id: 'addMappingWithTransform',
    label: 'Add Mapping Transform',
    icon: '🔄',
    description: 'Add mapping with transformation',
    category: 'etl'
  },
  {
    id: 'createTransform',
    label: 'Create Transform',
    icon: '⚡',
    description: 'Create data transformation',
    category: 'etl'
  },
  {
    id: 'doETL',
    label: 'Do ETL',
    icon: '🔄',
    description: 'Execute ETL process',
    category: 'etl'
  },
  {
    id: 'etlStatus',
    label: 'ETL Status',
    icon: '📊',
    description: 'Get ETL process status',
    category: 'etl'
  },
  {
    id: 'getTransform',
    label: 'Get Transform',
    icon: '📥',
    description: 'Retrieve transformation',
    category: 'etl'
  },
  {
    id: 'listTransforms',
    label: 'List Transforms',
    icon: '📋',
    description: 'List all transformations',
    category: 'etl'
  },
  {
    id: 'registerTransform',
    label: 'Register Transform',
    icon: '📝',
    description: 'Register new transformation',
    category: 'etl'
  },

  // Host Functions
  {
    id: 'callMethod',
    label: 'Call Method',
    icon: '📞',
    description: 'Call host method',
    category: 'host'
  },
  {
    id: 'getHostObject',
    label: 'Get Host Object',
    icon: '🖥️',
    description: 'Get host object reference',
    category: 'host'
  },
  {
    id: 'hostObject',
    label: 'Host Object',
    icon: '🔗',
    description: 'Access host object',
    category: 'host'
  },

  // JSON Functions
  {
    id: 'parseJSON',
    label: 'Parse JSON',
    icon: '📖',
    description: 'Parse a JSON string into a JSONNode',
    category: 'json'
  },
  {
    id: 'parseJSONSimple',
    label: 'Parse JSON Simple',
    icon: '📋',
    description: 'Parse a JSON string into a SimpleJSON node',
    category: 'json'
  },
  {
    id: 'toJSON',
    label: 'To JSON',
    icon: '📝',
    description: 'Convert a Chariot value or JSONNode to a JSON string',
    category: 'json'
  },
  {
    id: 'toSimpleJSON',
    label: 'To Simple JSON',
    icon: '📄',
    description: 'Convert a Chariot value to a SimpleJSON node',
    category: 'json'
  },

  // Node Functions
  {
    id: 'addChild',
    label: 'Add Child',
    icon: '➕',
    description: 'Add child node',
    category: 'node'
  },
  {
    id: 'childCount',
    label: 'Child Count',
    icon: '🔢',
    description: 'Count child nodes',
    category: 'node'
  },
  {
    id: 'clear',
    label: 'Clear',
    icon: '🧹',
    description: 'Clear node contents',
    category: 'node'
  },
  {
    id: 'create',
    label: 'Create',
    icon: '🆕',
    description: 'Create new node',
    category: 'node'
  },
  {
    id: 'csvNode',
    label: 'CSV Node',
    icon: '📊',
    description: 'Create CSV node',
    category: 'node'
  },
  {
    id: 'findByName',
    label: 'Find By Name',
    icon: '🔍',
    description: 'Find node by name',
    category: 'node'
  },
  {
    id: 'firstChild',
    label: 'First Child',
    icon: '⬆️',
    description: 'Get first child node',
    category: 'node'
  },
  {
    id: 'getAttribute',
    label: 'Get Attribute',
    icon: '🏷️',
    description: 'Get node attribute',
    category: 'node'
  },
  {
    id: 'getChildAt',
    label: 'Get Child At',
    icon: '📍',
    description: 'Get child at index',
    category: 'node'
  },
  {
    id: 'getChildByName',
    label: 'Get Child By Name',
    icon: '🔎',
    description: 'Get child by name',
    category: 'node'
  },
  {
    id: 'getDepth',
    label: 'Get Depth',
    icon: '📏',
    description: 'Get node depth',
    category: 'node'
  },
  {
    id: 'getLevel',
    label: 'Get Level',
    icon: '📶',
    description: 'Get node level',
    category: 'node'
  },
  {
    id: 'getName',
    label: 'Get Name',
    icon: '🔤',
    description: 'Get node name',
    category: 'node'
  },
  {
    id: 'getParent',
    label: 'Get Parent',
    icon: '⬆️',
    description: 'Get parent node',
    category: 'node'
  },
  {
    id: 'getPath',
    label: 'Get Path',
    icon: '🛤️',
    description: 'Get node path',
    category: 'node'
  },
  {
    id: 'getRoot',
    label: 'Get Root',
    icon: '🌳',
    description: 'Get root node',
    category: 'node'
  },
  {
    id: 'getSiblings',
    label: 'Get Siblings',
    icon: '👫',
    description: 'Get sibling nodes',
    category: 'node'
  },
  {
    id: 'getText',
    label: 'Get Text',
    icon: '📝',
    description: 'Get node text',
    category: 'node'
  },
  {
    id: 'hasAttribute',
    label: 'Has Attribute',
    icon: '❓',
    description: 'Check if has attribute',
    category: 'node'
  },
  {
    id: 'isLeaf',
    label: 'Is Leaf',
    icon: '🍃',
    description: 'Check if leaf node',
    category: 'node'
  },
  {
    id: 'isRoot',
    label: 'Is Root',
    icon: '🌿',
    description: 'Check if root node',
    category: 'node'
  },
  {
    id: 'jsonNode',
    label: 'JSON Node',
    icon: '📄',
    description: 'Create JSON node',
    category: 'node'
  },
  {
    id: 'lastChild',
    label: 'Last Child',
    icon: '⬇️',
    description: 'Get last child node',
    category: 'node'
  },
  {
    id: 'list',
    label: 'List',
    icon: '📋',
    description: 'List nodes',
    category: 'node'
  },
  {
    id: 'mapNode',
    label: 'Map Node',
    icon: '🗺️',
    description: 'Create map node',
    category: 'node'
  },
  {
    id: 'nodeToString',
    label: 'Node To String',
    icon: '📝',
    description: 'Convert node to string',
    category: 'node'
  },
  {
    id: 'queryNode',
    label: 'Query Node',
    icon: '🔍',
    description: 'Query node',
    category: 'node'
  },
  {
    id: 'removeAttribute',
    label: 'Remove Attribute',
    icon: '❌',
    description: 'Remove node attribute',
    category: 'node'
  },
  {
    id: 'removeChild',
    label: 'Remove Child',
    icon: '➖',
    description: 'Remove child node',
    category: 'node'
  },
  {
    id: 'setAttribute',
    label: 'Set Attribute',
    icon: '🏷️',
    description: 'Set node attribute',
    category: 'node'
  },
  {
    id: 'setAttributes',
    label: 'Set Attributes',
    icon: '🏷️',
    description: 'Set multiple attributes',
    category: 'node'
  },
  {
    id: 'setName',
    label: 'Set Name',
    icon: '✏️',
    description: 'Set node name',
    category: 'node'
  },
  {
    id: 'setText',
    label: 'Set Text',
    icon: '📝',
    description: 'Set node text',
    category: 'node'
  },
  {
    id: 'traverseNode',
    label: 'Traverse Node',
    icon: '🚶',
    description: 'Traverse node tree',
    category: 'node'
  },
  {
    id: 'xmlNode',
    label: 'XML Node',
    icon: '📄',
    description: 'Create XML node',
    category: 'node'
  },
  {
    id: 'yamlNode',
    label: 'YAML Node',
    icon: '📄',
    description: 'Create YAML node',
    category: 'node'
  },

  // CSV Functions
  {
    id: 'csvHeaders',
    label: 'csvHeaders',
    icon: '📋',
    description: 'Get the header row of a CSV file as an array of strings',
    category: 'csv'
  },
  {
    id: 'csvRowCount',
    label: 'csvRowCount',
    icon: '📊',
    description: 'Get the number of data rows in the CSV file (excluding the header row)',
    category: 'csv'
  },
  {
    id: 'csvColumnCount',
    label: 'csvColumnCount',
    icon: '📈',
    description: 'Get the number of columns in the CSV file',
    category: 'csv'
  },
  {
    id: 'csvGetRow',
    label: 'csvGetRow',
    icon: '📄',
    description: 'Get a specific row as a map where keys are column headers and values are cell values',
    category: 'csv'
  },
  {
    id: 'csvGetCell',
    label: 'csvGetCell',
    icon: '🔍',
    description: 'Get the value of a specific cell by row and column index or name',
    category: 'csv'
  },
  {
    id: 'csvGetRows',
    label: 'csvGetRows',
    icon: '📑',
    description: 'Get all data rows as an array of arrays. Use with caution on large CSV files',
    category: 'csv'
  },
  {
    id: 'csvToCSV',
    label: 'csvToCSV',
    icon: '🗂️',
    description: 'Convert a CSVNode back to a CSV-formatted string',
    category: 'csv'
  },
  {
    id: 'csvLoad',
    label: 'csvLoad',
    icon: '📥',
    description: 'Load a CSV file into an existing CSVNode instance',
    category: 'csv'
  },

  // SQL Functions
  {
    id: 'sqlBegin',
    label: 'SQL Begin',
    icon: '🚀',
    description: 'Begin SQL transaction',
    category: 'sql'
  },
  {
    id: 'sqlConnect',
    label: 'SQL Connect',
    icon: '🔌',
    description: 'Connect to SQL database',
    category: 'sql'
  },
  {
    id: 'sqlClose',
    label: 'SQL Close',
    icon: '🔚',
    description: 'Close SQL connection',
    category: 'sql'
  },
  {
    id: 'sqlCommit',
    label: 'SQL Commit',
    icon: '✅',
    description: 'Commit SQL transaction',
    category: 'sql'
  },
  {
    id: 'sqlExecute',
    label: 'SQL Execute',
    icon: '⚡',
    description: 'Execute SQL statement',
    category: 'sql'
  },
  {
    id: 'sqlListTables',
    label: 'SQL List Tables',
    icon: '📋',
    description: 'List database tables',
    category: 'sql'
  },
  {
    id: 'sqlQuery',
    label: 'SQL Query',
    icon: '🔍',
    description: 'Execute SQL query',
    category: 'sql'
  },
  {
    id: 'sqlRollback',
    label: 'SQL Rollback',
    icon: '↩️',
    description: 'Rollback SQL transaction',
    category: 'sql'
  },

  // Tree Functions
  {
    id: 'newTree',
    label: 'New Tree',
    icon: '🌳',
    description: 'Create new tree',
    category: 'tree'
  },
  {
    id: 'treeFind',
    label: 'Tree Find',
    icon: '🔍',
    description: 'Find in tree',
    category: 'tree'
  },
  {
    id: 'treeGetMetadata',
    label: 'Tree Get Metadata',
    icon: '📊',
    description: 'Get tree metadata',
    category: 'tree'
  },
  {
    id: 'treeLoad',
    label: 'Tree Load',
    icon: '📥',
    description: 'Load tree from file',
    category: 'tree'
  },
  {
    id: 'treeLoadSecure',
    label: 'Tree Load Secure',
    icon: '🔒',
    description: 'Load tree securely',
    category: 'tree'
  },
  {
    id: 'treeSave',
    label: 'Tree Save',
    icon: '💾',
    description: 'Save tree to file',
    category: 'tree'
  },
  {
    id: 'treeSaveSecure',
    label: 'Tree Save Secure',
    icon: '🔐',
    description: 'Save tree securely',
    category: 'tree'
  },
  {
    id: 'treeSearch',
    label: 'Tree Search',
    icon: '🔎',
    description: 'Search tree',
    category: 'tree'
  },
  {
    id: 'treeToYAML',
    label: 'Tree To YAML',
    icon: '📄',
    description: 'Convert tree to YAML',
    category: 'tree'
  },
  {
    id: 'treeToXML',
    label: 'Tree To XML',
    icon: '📄',
    description: 'Convert tree to XML',
    category: 'tree'
  },
  {
    id: 'treeValidateSecure',
    label: 'Tree Validate Secure',
    icon: '✅',
    description: 'Validate tree securely',
    category: 'tree'
  },
  {
    id: 'treeWalk',
    label: 'Tree Walk',
    icon: '🚶',
    description: 'Walk through tree',
    category: 'tree'
  },

  // File Functions
  {
    id: 'loadJSON',
    label: 'Load JSON',
    icon: '📂',
    description: 'Load JSON file',
    category: 'json'
  },
  {
    id: 'saveJSON',
    label: 'Save JSON',
    icon: '💾',
    description: 'Save JSON file',
    category: 'json'
  },
  {
    id: 'readFile',
    label: 'Read File',
    icon: '📖',
    description: 'Read file contents',
    category: 'file'
  },
  {
    id: 'writeFile',
    label: 'Write File',
    icon: '✍️',
    description: 'Write file contents',
    category: 'file'
  },
  {
    id: 'fileExists',
    label: 'File Exists',
    icon: '🔍',
    description: 'Check if file exists',
    category: 'file'
  },

  // Crypto Functions
  {
    id: 'encrypt',
    label: 'Encrypt',
    icon: '🔒',
    description: 'Encrypt data',
    category: 'crypto'
  },
  {
    id: 'decrypt',
    label: 'Decrypt',
    icon: '🔓',
    description: 'Decrypt data',
    category: 'crypto'
  },
  {
    id: 'hash256',
    label: 'Hash 256',
    icon: '#️⃣',
    description: 'SHA-256 hash',
    category: 'crypto'
  },
  {
    id: 'sign',
    label: 'Sign',
    icon: '✍️',
    description: 'Digital signature',
    category: 'crypto'
  },
  {
    id: 'cryptoRandomString',
    label: 'Crypto Random String',
    icon: '🛡️',
    description: 'Cryptographically secure random string',
    category: 'crypto'
  },

  // Math Functions
  {
    id: 'add',
    label: 'Add',
    icon: '➕',
    description: 'Addition operation',
    category: 'math'
  },
  {
    id: 'sub',
    label: 'Subtract',
    icon: '➖',
    description: 'Subtraction operation',
    category: 'math'
  },
  {
    id: 'mul',
    label: 'Multiply',
    icon: '✖️',
    description: 'Multiplication operation',
    category: 'math'
  },
  {
    id: 'div',
    label: 'Divide',
    icon: '➗',
    description: 'Division operation',
    category: 'math'
  },
  {
    id: 'abs',
    label: 'Absolute',
    icon: '📏',
    description: 'Absolute value',
    category: 'math'
  },
  {
    id: 'max',
    label: 'Maximum',
    icon: '⬆️',
    description: 'Maximum value',
    category: 'math'
  },
  {
    id: 'min',
    label: 'Minimum',
    icon: '⬇️',
    description: 'Minimum value',
    category: 'math'
  },
  {
    id: 'round',
    label: 'Round',
    icon: '🔄',
    description: 'Round number',
    category: 'math'
  },
  {
    id: 'random',
    label: 'Random',
    icon: '🎲',
    description: 'Random number',
    category: 'math'
  },
  {
    id: 'randomString',
    label: 'Random String',
    icon: '🔤',
    description: 'Pseudo-random alphanumeric string',
    category: 'math'
  },
  {
    id: 'transpose',
    label: 'Transpose',
    icon: '🔁',
    description: 'Transpose a matrix',
    category: 'math'
  },
  {
    id: 'matmul',
    label: 'Matmul',
    icon: '🧮',
    description: 'Matrix multiplication',
    category: 'math'
  },
  {
    id: 'solveLinear',
    label: 'Solve Linear',
    icon: '📐',
    description: 'Solve A·x = b systems',
    category: 'math'
  },
  {
    id: 'lsp',
    label: 'Least Squares',
    icon: '📊',
    description: 'Least-squares projection',
    category: 'math'
  },
  {
    id: 'vectorScale',
    label: 'Vector Scale',
    icon: '📐',
    description: 'Scale a vector by a scalar',
    category: 'math'
  },
  {
    id: 'dotProduct',
    label: 'Dot Product',
    icon: '🎯',
    description: 'Dot product of two vectors',
    category: 'math'
  },

  // String Functions
  {
    id: 'concat',
    label: 'Concat',
    icon: '🔗',
    description: 'Concatenate strings',
    category: 'string'
  },
  {
    id: 'split',
    label: 'Split',
    icon: '✂️',
    description: 'Split string',
    category: 'string'
  },
  {
    id: 'replace',
    label: 'Replace',
    icon: '🔄',
    description: 'Replace in string',
    category: 'string'
  },
  {
    id: 'substring',
    label: 'Substring',
    icon: '📝',
    description: 'Extract substring',
    category: 'string'
  },
  {
    id: 'stringLength',
    label: 'String Length',
    icon: '📏',
    description: 'String length',
    category: 'string'
  },
  {
    id: 'upper',
    label: 'Upper',
    icon: '🔤',
    description: 'Uppercase string',
    category: 'string'
  },
  {
    id: 'lower',
    label: 'Lower',
    icon: '🔡',
    description: 'Lowercase string',
    category: 'string'
  },

  // System Functions
  {
    id: 'logPrint',
    label: 'Log Print',
    icon: '📝',
    description: 'Print to log',
    category: 'system'
  },
  {
    id: 'sleep',
    label: 'Sleep',
    icon: '😴',
    description: 'Pause execution',
    category: 'system'
  },
  {
    id: 'getEnv',
    label: 'Get Env',
    icon: '🌐',
    description: 'Get environment variable',
    category: 'system'
  },
  {
    id: 'exit',
    label: 'Exit',
    icon: '🚪',
    description: 'Exit program',
    category: 'system'
  },

  // Value Functions
  {
    id: 'declare',
    label: 'Declare',
    icon: '📋',
    description: 'Declare variable',
    category: 'value'
  },
  {
    id: 'symbol',
    label: 'Symbol',
    icon: '🔣',
    description: 'Evaluate a variable by name (symbol(name))',
    category: 'value'
  },
  {
    id: 'setValue',
    label: 'Set Equal',
    icon: '💾',
    description: 'Assign a variable',
    category: 'value'
  },
  {
    id: 'valueOf',
    label: 'Value Of',
    icon: '📊',
    description: 'Get variable value',
    category: 'value'
  },
  {
    id: 'typeOf',
    label: 'Type Of',
    icon: '🏷️',
    description: 'Get variable type',
    category: 'value'
  },
  {
    id: 'exists',
    label: 'Exists',
    icon: '❓',
    description: 'Check if exists',
    category: 'value'
  },

  // Machine Learning / RL Functions
  {
    id: 'rlInit',
    label: 'RL Init',
    icon: '🤖',
    description: 'Initialize RL scorer with config',
    category: 'ml'
  },
  {
    id: 'rlScore',
    label: 'RL Score',
    icon: '📊',
    description: 'Score candidates using RL model',
    category: 'ml'
  },
  {
    id: 'rlLearn',
    label: 'RL Learn',
    icon: '🧠',
    description: 'Update RL model with feedback',
    category: 'ml'
  },
  {
    id: 'rlClose',
    label: 'RL Close',
    icon: '🔒',
    description: 'Release RL scorer resources',
    category: 'ml'
  },
  {
    id: 'rlSelectBest',
    label: 'RL Select Best',
    icon: '🏆',
    description: 'Select candidate with highest score',
    category: 'ml'
  },
  {
    id: 'extractRLFeatures',
    label: 'Extract RL Features',
    icon: '🔍',
    description: 'Extract feature vectors from candidates',
    category: 'ml'
  },
  {
    id: 'rlExplore',
    label: 'RL Explore',
    icon: '🎲',
    description: 'Epsilon-greedy exploration',
    category: 'ml'
  },
  {
    id: 'nbaDecision',
    label: 'NBA Decision',
    icon: '🎯',
    description: 'Complete Next-Best Action workflow',
    category: 'ml'
  }
];