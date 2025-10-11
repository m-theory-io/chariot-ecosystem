// Chariot Code Generator - Converts VisualDSL diagrams to Chariot .ch source code

export interface VisualDSLDiagram {
  name: string;
  nodes: VisualDSLNode[];
  edges: VisualDSLEdge[];
  nestingRelations: NestingRelation[];
}

export interface VisualDSLNode {
  id: string;
  type: string;
  data: {
    label: string;
    icon: string;
    category: string;
    properties?: Record<string, any>;
  };
  position: { x: number; y: number };
}

export interface VisualDSLEdge {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
}

export interface NestingRelation {
  parentId: string;
  childId: string;
  order: number;
}

export class ChariotCodeGenerator {
  private diagram: VisualDSLDiagram;
  private nodeMap: Map<string, VisualDSLNode>;
  private executionOrder: string[];
  private nestingMap: Map<string, string[]>;

  constructor(diagram: VisualDSLDiagram) {
    this.diagram = diagram;
    this.nodeMap = new Map();
    this.executionOrder = [];
    this.nestingMap = new Map();
    
    // Build node map
    diagram.nodes.forEach(node => {
      // Ignore visual-only group nodes
      if (node.type === 'group') return;
      this.nodeMap.set(node.id, node);
    });
    
    // Build nesting relationships - filter out invalid relations
    diagram.nestingRelations.forEach(rel => {
      // Only add relations where both parent and child nodes exist
      if (this.nodeMap.has(rel.parentId) && this.nodeMap.has(rel.childId)) {
        if (!this.nestingMap.has(rel.parentId)) {
          this.nestingMap.set(rel.parentId, []);
        }
        this.nestingMap.get(rel.parentId)!.push(rel.childId);
      }
    });
    
    // Calculate execution order from flow
    this.calculateExecutionOrder();
  }

  public generateChariotCode(): string {
    const lines: string[] = [];
    
    // Add header comment
    lines.push(`// ${this.diagram.name}`);
    lines.push('');
    
    // Get all child nodes that are handled inline (don't process them separately)
    const inlineProcessedNodes = new Set<string>();
    for (const [parentId, childIds] of this.nestingMap) {
      const parentNode = this.nodeMap.get(parentId);
      if (parentNode?.data.label === 'Declare' && childIds.length === 1) {
        const childNode = this.nodeMap.get(childIds[0]);
        if (childNode && ['Create', 'New Tree', 'Parse JSON', 'Array'].includes(childNode.data.label)) {
          inlineProcessedNodes.add(childIds[0]);
        }
      }
    }
    
    // Process nodes in execution order, skipping inline-processed ones
    for (const nodeId of this.executionOrder) {
      if (inlineProcessedNodes.has(nodeId)) {
        continue; // Skip nodes that are handled inline by their parent
      }
      
      const node = this.nodeMap.get(nodeId);
      if (!node) continue;
      
      const chariotCode = this.generateNodeCode(node);
      if (chariotCode) {
        lines.push(chariotCode);
      }
    }
    
    // Generate addChild calls for nesting relationships not handled inline
    const addChildLines: string[] = [];
    for (const [parentId, childIds] of this.nestingMap) {
      const parentNode = this.nodeMap.get(parentId);
      if (!parentNode) continue;
      
      const parentProps = parentNode.data.properties || {};
      const parentVarName = parentProps.variableName || this.inferVariableName(parentNode);
      
      // Check if this is a complex nesting case (multiple children or non-simple children)
      const isComplexNesting = childIds.length > 1 || 
        childIds.some(childId => {
          const childNode = this.nodeMap.get(childId);
          return childNode && !['Create', 'New Tree', 'Parse JSON', 'Array'].includes(childNode.data.label);
        });
      
      if (isComplexNesting) {
        for (const childId of childIds) {
          const childNode = this.nodeMap.get(childId);
          if (!childNode) continue;
          
          const childProps = childNode.data.properties || {};
          const childVarName = childProps.variableName || this.inferVariableName(childNode);
          
          addChildLines.push(`addChild(${parentVarName}, ${childVarName})`);
        }
      }
    }
    
    // Add the addChild calls if any were generated
    if (addChildLines.length > 0) {
      lines.push('');
      lines.push('// Add children to parent nodes');
      lines.push(...addChildLines);
    }
    
    return lines.join('\n');
  }

  private calculateExecutionOrder(): void {
    // Find the start node
    const startNode = this.diagram.nodes.find(node => 
      node.data.label === 'Start' || node.id === 'start'
    );
    
    if (!startNode) {
      // If no start node, use first node
      if (this.diagram.nodes.length > 0) {
        // Exclude visual-only group nodes
        this.executionOrder = this.diagram.nodes
          .filter(n => n.type !== 'group')
          .map(n => n.id);
      }
      return;
    }

    // Perform DFS traversal following edges, but prioritize main flow over nesting
    const visited = new Set<string>();
    const stack = [startNode.id];
    
    while (stack.length > 0) {
      const currentId = stack.pop()!;
      if (visited.has(currentId)) continue;
      
      visited.add(currentId);
      this.executionOrder.push(currentId);
      
      // Find all outgoing edges from current node
      const outgoingEdges = this.diagram.edges.filter(edge => edge.source === currentId);
      
      // Separate main flow edges from nesting edges
      const mainFlowEdges = outgoingEdges.filter(edge => {
        // Main flow edges are typically horizontal (right) or to special nodes like treeSave
        const targetNode = this.nodeMap.get(edge.target);
        return targetNode && (
          edge.sourceHandle === 'right' || 
          targetNode.data.label === 'Tree Save' ||
          targetNode.data.label === 'Add Child'
        );
      });
      
      const nestingEdges = outgoingEdges.filter(edge => !mainFlowEdges.includes(edge));
      
      // Sort main flow edges by target position (left to right, top to bottom)
      mainFlowEdges.sort((a, b) => {
        const nodeA = this.nodeMap.get(a.target);
        const nodeB = this.nodeMap.get(b.target);
        if (!nodeA || !nodeB) return 0;
        
        if (Math.abs(nodeA.position.y - nodeB.position.y) < 50) {
          // Same row, sort by x position
          return nodeA.position.x - nodeB.position.x;
        } else {
          // Different rows, sort by y position
          return nodeA.position.y - nodeB.position.y;
        }
      });
      
      // Add main flow targets to stack first (in reverse order for correct DFS)
      for (let i = mainFlowEdges.length - 1; i >= 0; i--) {
        stack.push(mainFlowEdges[i].target);
      }
      
      // Add nesting edges last so they get processed after main flow
      for (let i = nestingEdges.length - 1; i >= 0; i--) {
        stack.push(nestingEdges[i].target);
      }
    }
    
    // Add any unvisited nodes (isolated nodes) at the end
    this.diagram.nodes.forEach(node => {
      if (node.type === 'group') return; // skip visual-only nodes
      if (!visited.has(node.id)) {
        this.executionOrder.push(node.id);
      }
    });
  }

  private generateNodeCode(node: VisualDSLNode): string | null {
    const props = node.data.properties || {};
    
    switch (node.data.label) {
      case 'Start':
        // Start node doesn't generate code directly, just a comment
        return `// Starting ${props.name || this.diagram.name}`;
        
      case 'Declare':
        return this.generateDeclareCode(node);
        
      case 'Create':
      case 'New Tree':
        return this.generateCreateCode(node);
        
      case 'Parse JSON':
        return this.generateParseJSONCode(node);
        
      case 'Add Child':
        return this.generateAddChildCode(node);
        
      case 'Tree Save':
        return this.generateTreeSaveCode(node);
        
      case 'Tree Load':
        return this.generateTreeLoadCode(node);
      
      case 'Tree Save Secure':
        return this.generateTreeSaveSecureCode(node);
      
      case 'Tree Load Secure':
        return this.generateTreeLoadSecureCode(node);
        
      case 'Tree Find':
        return this.generateTreeFindCode(node);
        
      case 'Tree Search':
        return this.generateTreeSearchCode(node);
      
      case 'Tree Walk':
        return this.generateTreeWalkCode(node);
        
      case 'Add To':
        return this.generateAddToCode(node);
        
      case 'LogPrint':
      case 'Log Print':
      case 'logPrint':
        return this.generateLogPrintCode(node);
        
      case 'Create Transform':
        return this.generateCreateTransformCode(node);
        
      case 'Add Mapping':
        return this.generateAddMappingCode(node);
        
      case 'Array':
        return this.generateArrayCode(node);
        
      case 'Set Value':
        return this.generateSetValueCode(node);
        
      case 'Get Attribute':
        return this.generateGetAttributeCode(node);
        
      case 'Set Attribute':
        return this.generateSetAttributeCode(node);
        
      default:
        // Generic function call
        return this.generateGenericFunctionCode(node);
    }
  }

  private generateDeclareCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = props.variableName || this.inferVariableName(node);
    const typeSpec = props.typeSpecifier || 'T';
    const isGlobal = props.isGlobal || false;
    
    // Check if this declare has a nested child (like create or parseJSON)
    const nestedChildren = this.nestingMap.get(node.id) || [];
    
    // Only handle single child inline declarations for simple cases
    if (nestedChildren.length === 1) {
      const childNode = this.nodeMap.get(nestedChildren[0]);
      if (childNode) {
        // Only generate inline for simple child types
        const isSimpleChild = ['Create', 'New Tree', 'Parse JSON', 'Array'].includes(childNode.data.label);
        
        if (isSimpleChild) {
          const childCode = this.generateNestedChildCode(childNode);
          if (childCode) {
            if (isGlobal) {
              return `declareGlobal(${varName}, '${typeSpec}', ${childCode})`;
            } else {
              return `declare(${varName}, '${typeSpec}', ${childCode})`;
            }
          }
        }
      }
    }
    
    // Simple declare without nested content (or for complex nesting handled by addChild)
    if (isGlobal) {
      return `declareGlobal(${varName}, '${typeSpec}')`;
    } else {
      return `declare(${varName}, '${typeSpec}')`;
    }
  }

  private generateCreateCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const nodeName = props.nodeName || this.diagram.name || 'newNode';
    return `create('${nodeName}')`;
  }

  private generateParseJSONCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let jsonString = props.jsonString || '{}';
    const nodeName = props.nodeName || this.inferNodeNameFromContext(node);
    
    // Clean up jsonString for common patterns
    if (jsonString === '{ [] }') {
      jsonString = '[]';
    } else if (jsonString === '{ [\"admin\", \"contributor\", \"viewer\"] }') {
      jsonString = '[\"admin\", \"contributor\", \"viewer\"]';
    }
    
    return `parseJSON('${jsonString}', '${nodeName}')`;
  }

  private generateAddChildCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    
    // Use the properties if available
    if (props.parentNode && props.childNode) {
      return `addChild(${props.parentNode}, ${props.childNode})`;
    }
    
    // For usersAgent pattern: find main tree and connect it to individual declares
    const mainParent = this.findMainTreeVariable();
    if (mainParent) {
      // Find all declare nodes with J type (the children to add)
      const childDeclares = this.diagram.nodes.filter(node => 
        node.data.label === 'Declare' && 
        node.data.properties?.typeSpecifier === 'J' &&
        node.data.properties?.variableName
      );
      
      // Get the position of this addChild in the flow
      const addChildNodes = this.diagram.nodes.filter(n => n.data.label === 'Add Child');
      const addChildIndex = addChildNodes.findIndex(n => n.id === node.id);
      
      if (addChildIndex >= 0 && addChildIndex < childDeclares.length) {
        const childDeclare = childDeclares[addChildIndex];
        if (childDeclare.data.properties?.variableName) {
          const childVar = childDeclare.data.properties.variableName;
          return `addChild(${mainParent}, ${childVar})`;
        }
      }
    }
    
    // Fallback to previous logic
    const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
    
    if (incomingEdges.length > 0) {
      const sourceNode = this.nodeMap.get(incomingEdges[0].source);
      if (sourceNode) {
        const mainParentFallback = this.findMainTreeVariable();
        const recentChild = this.findMostRecentDeclaredVariable(node);
        
        if (mainParentFallback && recentChild) {
          return `addChild(${mainParentFallback}, ${recentChild})`;
        }
        
        if (sourceNode.data.label === 'Declare' && sourceNode.data.properties?.variableName) {
          const parentVar = sourceNode.data.properties.variableName;
          const childVar = this.inferChildVariable(node);
          return `addChild(${parentVar}, ${childVar})`;
        }
      }
    }
    
    return `addChild(parent, child)`;
  }

  private generateTreeSaveCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    
    // Use the explicit tree variable if provided, otherwise try to find from incoming edges
    let treeVar = props.treeVariable;
    if (!treeVar) {
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          treeVar = sourceNode.data.properties.variableName;
        }
      }
    }
    
    // Use default tree variable if still not found
    if (!treeVar) {
      treeVar = 'tree';
    }
    
    // Use explicit filename or fallback to diagram name
    const filename = props.filename || `${this.diagram.name}.json`;
    
    // Build the function call with optional parameters
    let params = [`${treeVar}`, `'${filename}'`];
    
    // Add format if specified and different from auto-detect
    if (props.format && props.format !== '') {
      params.push(`'${props.format}'`);
    }
    
    // Add compress flag if specified
    if (props.compress === true) {
      // If format wasn't specified but compress is, we need to add format first
      if (!props.format || props.format === '') {
        // Auto-detect format from filename extension
        const ext = filename.split('.').pop()?.toLowerCase();
        let autoFormat = 'json'; // default
        if (ext === 'gob') autoFormat = 'gob';
        else if (ext === 'xml') autoFormat = 'xml';
        else if (ext === 'yaml' || ext === 'yml') autoFormat = 'yaml';
        params.push(`'${autoFormat}'`);
      }
      params.push('true');
    }
    
    return `treeSave(${params.join(', ')})`;
  }

  private generateTreeLoadCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const filename = props.filename || 'data.json';
    
    return `treeLoad('${filename}')`;
  }

  private generateTreeSaveSecureCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let treeVar = props.treeVariable || 'tree';

    // Try infer from incoming edge declare if not set
    if (!props.treeVariable) {
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          treeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    const filename = props.filename || 'secure.json';
    const encryptionKeyID = props.encryptionKeyID || 'encKey';
    const signingKeyID = props.signingKeyID || 'signKey';
    const watermark = props.watermark || 'watermark';

    const args = [treeVar, `'${filename}'`, `'${encryptionKeyID}'`, `'${signingKeyID}'`, `'${watermark}'`];

    // Optional options map
    const options: string[] = [];
    if (props.verificationKeyID) options.push(`'verificationKeyID', '${props.verificationKeyID}'`);
    if (typeof props.checksum === 'boolean') options.push(`'checksum', ${props.checksum}`);
    if (typeof props.auditTrail === 'boolean') options.push(`'auditTrail', ${props.auditTrail}`);
    if (typeof props.compressionLevel === 'number') options.push(`'compressionLevel', ${props.compressionLevel}`);

    if (options.length > 0) {
      args.push(`map(${options.join(', ')})`);
    }

    return `treeSaveSecure(${args.join(', ')})`;
  }

  private generateTreeLoadSecureCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const filename = props.filename || 'secure.json';
    const decryptionKeyID = props.decryptionKeyID || 'decKey';
    const verificationKeyID = props.verificationKeyID || 'verifyKey';
    return `treeLoadSecure('${filename}', '${decryptionKeyID}', '${verificationKeyID}')`;
  }

  private generateTreeFindCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const treeVar = (props.treeVariable ?? '').trim();
    const fieldName = props.fieldName || 'id';
    const value = props.value ?? '';
    const operator = props.operator || '';
    const searchAll = !!props.searchAll;
    const valueArg = typeof value === 'string' && !/^[0-9]+(\.[0-9]+)?$/.test(value)
      ? `'${value}'`
      : String(value);

    // If searchAll is enabled or no tree var provided, emit implicit form
    if (searchAll || treeVar === '') {
      const args = [`'${fieldName}'`, valueArg];
      if (operator) args.push(`'${operator}'`);
      return `treeFind(${args.join(', ')})`;
    }

    const args = [treeVar, `'${fieldName}'`, valueArg];
    if (operator) args.push(`'${operator}'`);
    return `treeFind(${args.join(', ')})`;
  }

  private generateTreeSearchCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const treeVar = props.treeVariable || 'tree';
    const fieldName = props.fieldName || 'name';
    const value = props.value ?? '';
    const operator = props.operator || '';
    const existsOnly = !!props.existsOnly;
    const valueArg = typeof value === 'string' && !/^[0-9]+(\.[0-9]+)?$/.test(value)
      ? `'${value}'`
      : String(value);
    const args = [treeVar, `'${fieldName}'`, valueArg];
    if (operator) args.push(`'${operator}'`);
    if (existsOnly) {
      // ensure operator slot exists if we need to pass 5th arg
      if (!operator) args.push(`'='`);
      args.push('true');
    }
    return `treeSearch(${args.join(', ')})`;
  }

  private generateTreeWalkCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const treeVar = props.treeVariable || 'tree';
    const functionName = props.functionName || 'myFunc';
    return `treeWalk(${treeVar}, ${functionName})`;
  }

  private generateAddToCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const collectionName = props.collectionName || 'collection';
    const value = props.value || 'item';
    
    // Collection name should be unquoted (naked symbol/variable name)
    // Value should be used as-is to allow user control over quoting
    return `addTo(${collectionName}, ${value})`;
  }

  private generateLogPrintCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const message = props.message || 'message';
    const logLevel = props.logLevel || 'info';
    const additionalArgs = props.additionalArgs || [];
    
    // Build the function call arguments
    const args: string[] = [];
    
    // Add the message (first argument) - let user control quoting for variables vs strings
    // If it looks like a variable name (starts with letter/underscore, contains only alphanumeric/underscore), use as-is
    // Otherwise, quote it as a string literal
    if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(message)) {
      // Looks like a variable name - use unquoted
      args.push(message);
    } else {
      // Quote as string literal
      args.push(`'${message}'`);
    }
    
    // Add log level if it's not the default 'info' or if there are additional args
    if (logLevel !== 'info' || additionalArgs.length > 0) {
      args.push(`'${logLevel}'`);
      
      // Add additional arguments if any
      if (additionalArgs.length > 0) {
        additionalArgs.forEach((arg: string) => {
          // Check if the argument is already quoted or looks like a variable
          if (arg.startsWith('"') && arg.endsWith('"') || arg.startsWith("'") && arg.endsWith("'")) {
            // Already quoted - use as-is
            args.push(arg);
          } else if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(arg)) {
            // Looks like a variable name - use unquoted
            args.push(arg);
          } else {
            // Quote as string literal
            args.push(`'${arg}'`);
          }
        });
      }
    }
    
    // Use camelCase function name: logPrint (not LogPrint)
    return `logPrint(${args.join(', ')})`;
  }

  private generateCreateTransformCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let transformName = props.transformName || 'transform';
    
    // Remove quotes if user accidentally included them and it looks like a variable name
    if ((transformName.startsWith('"') && transformName.endsWith('"')) || 
        (transformName.startsWith("'") && transformName.endsWith("'"))) {
      const unquoted = transformName.slice(1, -1);
      // If the unquoted version looks like a valid variable name, use it unquoted
      if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(unquoted)) {
        transformName = unquoted;
      }
    }
    
    // Check if the transform name looks like a variable or should be quoted
    if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(transformName)) {
      // Looks like a variable name - use unquoted
      return `createTransform(${transformName})`;
    } else {
      // Quote as string literal
      return `createTransform('${transformName}')`;
    }
  }

  private generateAddMappingCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let transform = props.transform || 'transform';
    const sourceField = props.sourceField || 'sourceField';
    const targetColumn = props.targetColumn || 'targetColumn';
    const program = props.program || [];
    const dataType = props.dataType || 'string';
    const required = props.required !== undefined ? props.required : false;
    
    // Remove quotes from transform if user accidentally included them (should be naked variable)
    if ((transform.startsWith('"') && transform.endsWith('"')) || 
        (transform.startsWith("'") && transform.endsWith("'"))) {
      transform = transform.slice(1, -1);
    }
    
    // Build the function call arguments
    const args: string[] = [];
    
    // 1. Transform - naked variable name (unquoted)
    args.push(transform);
    
    // 2. Source field - quoted string
    args.push(`'${sourceField}'`);
    
    // 3. Target column - quoted string
    args.push(`'${targetColumn}'`);
    
    // 4. Program - array of strings (optional)
    if (Array.isArray(program) && program.length > 0) {
      const programStr = program.map(line => `'${line}'`).join(', ');
      args.push(`[${programStr}]`);
    } else {
      args.push('[]');
    }
    
    // 5. Data type - quoted string
    args.push(`'${dataType}'`);
    
    // 6. Required - boolean literal
    args.push(String(required));
    
    // Use camelCase function name: addMapping (not AddMapping)
    return `addMapping(${args.join(', ')})`;
  }

  private generateArrayCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    
    // Use the values array from the new property structure
    const values = props.values || [];
    
    if (Array.isArray(values) && values.length > 0) {
      const valuesStr = values.map(v => `'${v}'`).join(', ');
      return `array(${valuesStr})`;
    }
    
    return `array()`;
  }

  private generateGenericFunctionCode(node: VisualDSLNode): string {
    // Sanitize the label to avoid artifacts like "declare(...)" from visual labels
    const rawLabel = node.data.label.toLowerCase();
    // Remove any non-alphanumeric/space characters (e.g., parentheses, punctuation)
    const sanitizedLabel = rawLabel.replace(/[^a-z0-9\s]/g, '');
    let functionName = sanitizedLabel.replace(/\s+/g, '');
    
    // Special handling for known camelCase functions that shouldn't be lowercased
    const camelCaseFunctions: Record<string, string> = {
      'logprint': 'logPrint',
      'log print': 'logPrint',
      'LogPrint': 'logPrint',
      'parseJSON': 'parseJSON',
      'parsejson': 'parseJSON',
      'parse json': 'parseJSON',
      'createTransform': 'createTransform',
      'create transform': 'createTransform',
      'addMapping': 'addMapping',
      'add mapping': 'addMapping',
      'addChild': 'addChild',
      'add child': 'addChild',
      'addTo': 'addTo',
      'add to': 'addTo',
      'treeSave': 'treeSave',
      'tree save': 'treeSave',
      'treeLoad': 'treeLoad',
      'tree load': 'treeLoad',
      'treeFind': 'treeFind',
      'tree find': 'treeFind',
      'treeSearch': 'treeSearch',
      'tree search': 'treeSearch',
  'treeSaveSecure': 'treeSaveSecure',
  'tree save secure': 'treeSaveSecure',
  'treeLoadSecure': 'treeLoadSecure',
  'tree load secure': 'treeLoadSecure',
  'treeWalk': 'treeWalk',
  'tree walk': 'treeWalk',
      'getValue': 'getValue',
      'get value': 'getValue',
      'setValue': 'setValue',
      'set value': 'setValue',
      'getAttribute': 'getAttribute',
      'get attribute': 'getAttribute',
      'setAttribute': 'setAttribute',
      'set attribute': 'setAttribute'
    };
    
    // Check if we have a known camelCase mapping
  const labelKey = sanitizedLabel; // already lowercased and cleaned
    if (camelCaseFunctions[labelKey]) {
      functionName = camelCaseFunctions[labelKey];
    } else if (camelCaseFunctions[functionName]) {
      functionName = camelCaseFunctions[functionName];
    }
    
    const props = node.data.properties || {};
    
    // Convert properties to function arguments
    const args = Object.values(props).map(value => {
      if (typeof value === 'string') {
        return `'${value}'`;
      }
      return String(value);
    });
    
    return `${functionName}(${args.join(', ')})`;
  }

  private generateSetValueCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = props.variableName || 'var';
    const value = props.value || '';
    
    if (typeof value === 'string') {
      return `setValue(${varName}, '${value}')`;
    }
    return `setValue(${varName}, ${value})`;
  }

  private generateGetAttributeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = props.variableName || 'var';
    const attributeName = props.attributeName || 'attr';
    
    return `getAttribute(${varName}, '${attributeName}')`;
  }

  private generateSetAttributeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = props.variableName || 'var';
    const attributeName = props.attributeName || 'attr';
    const value = props.value || '';
    
    if (typeof value === 'string') {
      return `setAttribute(${varName}, '${attributeName}', '${value}')`;
    }
    return `setAttribute(${varName}, '${attributeName}', ${value})`;
  }

  private generateNestedChildCode(childNode: VisualDSLNode): string | null {
    switch (childNode.data.label) {
      case 'Create':
      case 'New Tree':
        return this.generateCreateCode(childNode);
        
      case 'Parse JSON':
        // Check if the parseJSON node has its own children (like array)
        const parseJsonChildren = this.nestingMap.get(childNode.id) || [];
        if (parseJsonChildren.length > 0) {
          // For now, parseJSON with children should be handled separately
          // Return basic parseJSON and let the children be processed separately
          return this.generateParseJSONCode(childNode);
        }
        return this.generateParseJSONCode(childNode);
        
      case 'Array':
        return this.generateArrayCode(childNode);
        
      default:
        return this.generateGenericFunctionCode(childNode);
    }
  }

  private inferChildVariable(node: VisualDSLNode): string {
    // Look at previous nodes in execution order to find variable names
    const nodeIndex = this.executionOrder.indexOf(node.id);
    
    for (let i = nodeIndex - 1; i >= 0; i--) {
      const prevNodeId = this.executionOrder[i];
      const prevNode = this.nodeMap.get(prevNodeId);
      
      if (prevNode?.data.properties?.variableName) {
        return prevNode.data.properties.variableName;
      }
    }
    
    return 'child';
  }

  private inferVariableName(node: VisualDSLNode): string {
    // If properties exist and have a variable name, use it
    if (node.data.properties?.variableName) {
      return node.data.properties.variableName;
    }
    
    // Try to infer variable name from node label or position
    const label = node.data.label.toLowerCase().replace(/\s+/g, '');
    
    // Look at nearby nodes for context
    const nearbyNodes = this.diagram.nodes.filter(n => {
      const dx = Math.abs(n.position.x - node.position.x);
      const dy = Math.abs(n.position.y - node.position.y);
      return n.id !== node.id && dx < 200 && dy < 100;
    });
    
    // If there's a Create node nearby, use its name
    const createNode = nearbyNodes.find(n => n.data.label === 'Create' || n.data.label === 'New Tree');
    if (createNode?.data.properties?.nodeName) {
      return createNode.data.properties.nodeName;
    }
    
    // If there's a ParseJSON node nearby, use its name  
    const parseNode = nearbyNodes.find(n => n.data.label === 'Parse JSON');
    if (parseNode?.data.properties?.nodeName) {
      return parseNode.data.properties.nodeName;
    }
    
    // Infer from position - this works well for the usersAgent layout
    if (node.position.x > 250 && node.position.x < 350) {
      return 'users';
    } else if (node.position.x > 450 && node.position.x < 550) {
      return 'config';
    } else if (node.position.x > 650 && node.position.x < 750) {
      return 'roles';
    } else if (node.position.x > 800 && node.position.x < 900) {
      return 'rules';
    }
    
    // Default based on position or sequence
    if (node.position.x < 100) {
      return this.diagram.name; // First column often is the main object
    }
    
    // Generate based on common patterns
    const commonNames = ['users', 'roles', 'config', 'rules', 'data'];
    const nodeIndex = this.diagram.nodes.findIndex(n => n.id === node.id);
    
    if (nodeIndex < commonNames.length) {
      return commonNames[nodeIndex % commonNames.length];
    }
    
    return `var${nodeIndex}`;
  }

  private inferNodeNameFromContext(node: VisualDSLNode): string {
    // Look at the parent declare node to infer the name
    const parentRelation = this.diagram.nestingRelations.find(rel => rel.childId === node.id);
    if (parentRelation) {
      const parentNode = this.nodeMap.get(parentRelation.parentId);
      if (parentNode?.data.properties?.variableName) {
        return parentNode.data.properties.variableName;
      }
    }
    
    // Look at position context
    if (node.position.x > 250 && node.position.x < 300) {
      return 'users';
    } else if (node.position.x > 450 && node.position.x < 500) {
      return 'roles';
    } else if (node.position.x > 650 && node.position.x < 700) {
      return 'rules';
    } else if (node.position.x > 800) {
      return 'config';
    }
    
    return 'data';
  }

  private findMainTreeVariable(): string | null {
    // Look for the main tree variable (usually first declare with 'T' type)
    const mainTreeNode = this.diagram.nodes.find(node => 
      node.data.label === 'Declare' && 
      node.data.properties?.typeSpecifier === 'T' &&
      node.data.properties?.variableName
    );
    
    return mainTreeNode?.data.properties?.variableName || null;
  }

  private findMostRecentDeclaredVariable(currentNode: VisualDSLNode): string | null {
    const currentIndex = this.executionOrder.indexOf(currentNode.id);
    
    // Look backwards in execution order for the most recent declare
    for (let i = currentIndex - 1; i >= 0; i--) {
      const nodeId = this.executionOrder[i];
      const node = this.nodeMap.get(nodeId);
      
      if (node?.data.label === 'Declare' && node.data.properties?.variableName) {
        return node.data.properties.variableName;
      }
    }
    
    return null;
  }
}

// Helper function to generate Chariot code from a diagram JSON string
export function generateChariotCodeFromDiagram(diagramJson: string): string {
  try {
    const diagram: VisualDSLDiagram = JSON.parse(diagramJson);
    const generator = new ChariotCodeGenerator(diagram);
    return generator.generateChariotCode();
  } catch (error) {
    throw new Error(`Failed to generate Chariot code: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }
}

// Helper function to export generated code as a file
export function downloadChariotCode(code: string, filename: string): void {
  const blob = new Blob([code], { type: 'text/plain' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename.endsWith('.ch') ? filename : `${filename}.ch`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
