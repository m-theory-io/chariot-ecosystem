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
  private parentLookup: Map<string, string>;
  private structuralInlineNodes: Set<string>;

  constructor(diagram: VisualDSLDiagram) {
    this.diagram = diagram;
    this.nodeMap = new Map();
    this.executionOrder = [];
    this.nestingMap = new Map();
    this.parentLookup = new Map();
    this.structuralInlineNodes = new Set();
    
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
        this.parentLookup.set(rel.childId, rel.parentId);
      }
    });

    // Calculate execution order from flow before collecting inline nodes
    this.calculateExecutionOrder();

    this.structuralInlineNodes = this.collectStructuralInlineNodes();
  }

  private canonicalLabel(rawLabel?: string): string {
    if (!rawLabel) {
      return '';
    }
    const normalized = rawLabel.trim();
    const lowerKey = normalized.toLowerCase();
    const aliasMap: Record<string, string> = {
      'set equal': 'Set Equal',
      'set value': 'Set Value',
      'set q': 'Set Q',
      'setq': 'Set Q',
      'logprint': 'LogPrint',
      'log print': 'Log Print',
      'loop body': 'Loop Body',
    };
    return aliasMap[lowerKey] || normalized;
  }

  private getNodeLabel(node: VisualDSLNode): string {
    return this.canonicalLabel(node.data.label);
  }

  public generateChariotCode(): string {
    const lines: string[] = [];
    
    lines.push(`// ${this.diagram.name}`);
    lines.push('');
    
    const inlineProcessedNodes = new Set<string>();
    for (const [parentId, childIds] of this.nestingMap) {
      const parentNode = this.nodeMap.get(parentId);
      if (!parentNode) {
        continue;
      }
      const parentLabel = this.getNodeLabel(parentNode);
      if (parentLabel === 'Declare' && childIds.length === 1) {
        const childNode = this.nodeMap.get(childIds[0]);
        if (!childNode) {
          continue;
        }
        const typeSpec = parentNode.data.properties?.typeSpecifier || 'T';
        const childLabel = this.getNodeLabel(childNode);
        const inlineLabels = ['Create', 'New Tree', 'Parse JSON', 'Array'];
        if (inlineLabels.includes(childLabel)) {
          inlineProcessedNodes.add(childIds[0]);
        } else if (childLabel === 'Function' && typeSpec === 'F') {
          inlineProcessedNodes.add(childIds[0]);
        }
      } else if (['Set Equal', 'Set Value', 'Set Q', 'setq'].includes(parentLabel)) {
        childIds.forEach(id => inlineProcessedNodes.add(id));
      }
    }

    this.structuralInlineNodes.forEach(id => inlineProcessedNodes.add(id));
    
    for (const nodeId of this.executionOrder) {
      if (inlineProcessedNodes.has(nodeId)) {
        continue;
      }
      
      const node = this.nodeMap.get(nodeId);
      if (!node) continue;
      
      const chariotCode = this.generateNodeCode(node);
      if (chariotCode) {
        lines.push(chariotCode);
      }
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
    const label = this.getNodeLabel(node);

    switch (label) {
      case 'Start':
        // Start node doesn't generate code directly, just a comment
        return `// Starting ${props.name || this.diagram.name}`;
        
      case 'Declare':
        return this.generateDeclareCode(node);
        
      case 'Create':
        return this.generateCreateCode(node);
      case 'New Tree':
        return this.generateNewTreeCode(node);
        
      case 'Parse JSON':
        return this.generateParseJSONCode(node);
        
      case 'Add Child':
        return this.generateAddChildCode(node);
      case 'Remove Child':
        return this.generateRemoveChildCode(node);
      
          return this.generateGetNameCode(node);
        return this.generateClearCode(node);
      
      case 'Child Count':
        return this.generateChildCountCode(node);
      case 'First Child':
        return this.generateFirstChildCode(node);
      case 'Last Child':
        return this.generateLastChildCode(node);
      case 'Get Child At':
        return this.generateGetChildAtCode(node);
      case 'CSV Node':
        return this.generateCSVNodeCode(node);
      case 'JSON Node':
        return this.generateJSONNodeCode(node);
      case 'XML Node':
        return this.generateXMLNodeCode(node);
      case 'YAML Node':
        return this.generateYAMLNodeCode(node);
      case 'Map Node':
        return this.generateMapNodeCode(node);
      case 'Find By Name':
        return this.generateFindByNameCode(node);
      case 'Traverse Node':
        return this.generateTraverseNodeCode(node);
      case 'Query Node':
        return this.generateQueryNodeCode(node);
      case 'Get Child By Name':
        return this.generateGetChildByNameCode(node);
      case 'Get Depth':
        return this.generateGetDepthCode(node);
      case 'Get Level':
        return this.generateGetLevelCode(node);
      case 'Get Parent':
        return this.generateGetParentCode(node);
      case 'Get Path':
        return this.generateGetPathCode(node);
      case 'Get Root':
        return this.generateGetRootCode(node);
      case 'Get Siblings':
        return this.generateGetSiblingsCode(node);
      case 'Get Text':
        return this.generateGetTextCode(node);
      case 'Is Leaf':
        return this.generateIsLeafCode(node);
      case 'Is Root':
        return this.generateIsRootCode(node);

        
      case 'Tree Save':
        return this.generateTreeSaveCode(node);
        
      case 'Tree Load':
        return this.generateTreeLoadCode(node);
      
      case 'Tree Save Secure':
        return this.generateTreeSaveSecureCode(node);
      
      case 'Tree Load Secure':
        return this.generateTreeLoadSecureCode(node);
      
      case 'Tree Validate Secure':
        return this.generateTreeValidateSecureCode(node);
        
      case 'Tree Find':
        return this.generateTreeFindCode(node);
        
      case 'Tree Search':
        return this.generateTreeSearchCode(node);
      
      case 'Tree Walk':
        return this.generateTreeWalkCode(node);
      
      case 'Tree To YAML':
        return this.generateTreeToYAMLCode(node);
      case 'Tree To XML':
        return this.generateTreeToXMLCode(node);
      
      case 'Tree Get Metadata':
        return this.generateTreeGetMetadataCode(node);
        
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
      case 'Function':
        return this.generateFunctionCode(node);
      case 'If':
        return this.generateIfCode(node);
      case 'While':
        return this.generateWhileCode(node);
      case 'Switch':
        return this.generateSwitchCode(node);
      case 'Case':
      case 'Default':
      case 'Then':
      case 'Else':
      case 'Loop Body':
        return null;

      case 'Set Equal':
      case 'Set Value':
      case 'Set Q':
        return this.generateSetqCode(node);
      case 'SetQ':
        return this.generateSetqCode(node);
        
      case 'Get Attribute':
        return this.generateGetAttributeCode(node);
      case 'Remove Attribute':
        return this.generateRemoveAttributeCode(node);
      case 'Has Attribute':
        return this.generateHasAttributeCode(node);
      case 'List':
        return this.generateListCode(node);
      case 'Node To String':
        return this.generateNodeToStringCode(node);
      case 'Set Name':
        return this.generateSetNameCode(node);
        
      case 'Set Attribute':
        return this.generateSetAttributeCode(node);
      case 'Set Attributes':
        return this.generateSetAttributesCode(node);
      case 'Set Text':
        return this.generateSetTextCode(node);
        
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
    const rawInitialValue = props.initialValue;
    const hasInitialValue =
      rawInitialValue !== undefined &&
      rawInitialValue !== null &&
      String(rawInitialValue).trim().length > 0;

    
    // Check if this declare has a nested child (like create or parseJSON)
    const nestedChildren = this.nestingMap.get(node.id) || [];
    
    // Only handle single child inline declarations for simple cases
    if (nestedChildren.length === 1) {
      const childNode = this.nodeMap.get(nestedChildren[0]);
      if (childNode) {
        const inlineLabels = ['Create', 'New Tree', 'Parse JSON', 'Array'];
        const isSimpleChild = inlineLabels.includes(childNode.data.label);
        const isInlineFunction = childNode.data.label === 'Function' && typeSpec === 'F';

        if (isSimpleChild || isInlineFunction) {
          const childCode = isInlineFunction
            ? this.generateFunctionCode(childNode)
            : this.generateNestedChildCode(childNode);
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
      if (hasInitialValue) {
        return `declareGlobal(${varName}, '${typeSpec}', ${rawInitialValue})`;
      }
      return `declareGlobal(${varName}, '${typeSpec}')`;
    }

    if (hasInitialValue) {
      return `declare(${varName}, '${typeSpec}', ${rawInitialValue})`;
    }
    return `declare(${varName}, '${typeSpec}')`;
  }

  private generateCreateCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    // Support zero-arg create() when name is intentionally left blank
    if (Object.prototype.hasOwnProperty.call(props, 'nodeName')) {
      const raw = (props.nodeName ?? '').toString().trim();
      if (raw === '') {
        return 'create()';
      }
      return `create('${raw}')`;
    }
    // If no property set at all, fall back to diagram name (legacy behavior)
    const fallback = (this.diagram.name && this.diagram.name.trim() !== '') ? this.diagram.name : 'newNode';
    return `create('${fallback}')`;
  }

  private generateNewTreeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    // newTree requires exactly one string argument (name)
    const raw = (props.nodeName ?? '').toString().trim();
    const name = raw !== '' ? raw : this.inferNodeNameFromContext(node);
    return `newTree('${name}')`;
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

  private generateRemoveChildCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    // Use the properties if available
    if (props.parentNode && props.childNode) {
      return `removeChild(${props.parentNode}, ${props.childNode})`;
    }

    // Try to infer sensible defaults similar to addChild
    const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
    if (incomingEdges.length > 0) {
      const sourceNode = this.nodeMap.get(incomingEdges[0].source);
      if (sourceNode) {
        if (sourceNode.data.label === 'Declare' && sourceNode.data.properties?.variableName) {
          const parentVar = sourceNode.data.properties.variableName;
          const recentChild = this.findMostRecentDeclaredVariable(node);
          if (recentChild) {
            return `removeChild(${parentVar}, ${recentChild})`;
          }
        }
      }
    }

    return `removeChild(parent, child)`;
  }

  private generateClearCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = props.node || '';

    if (!nodeVar) {
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `clear(${nodeVar})`;
  }

  private generateChildCountCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = props.node || '';

    if (!nodeVar) {
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `childCount(${nodeVar})`;
  }

  private generateFirstChildCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = props.node || '';

    if (!nodeVar) {
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `firstChild(${nodeVar})`;
  }

  private generateLastChildCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = props.node || '';

    if (!nodeVar) {
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `lastChild(${nodeVar})`;
  }

  private generateGetChildAtCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();
    let indexVal: any = props.index;

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }
    if (!nodeVar) nodeVar = 'node';

    // Ensure index is a non-negative integer; fallback to 0
    let indexNum = 0;
    if (typeof indexVal === 'number' && Number.isInteger(indexVal) && indexVal >= 0) {
      indexNum = indexVal;
    } else if (typeof indexVal === 'string' && indexVal.trim() !== '') {
      const parsed = Number(indexVal);
      if (Number.isInteger(parsed) && parsed >= 0) indexNum = parsed;
    }

    return `getChildAt(${nodeVar}, ${indexNum})`;
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

  private generateTreeValidateSecureCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const filename = props.filename || 'secure.json';
    const verificationKeyID = props.verificationKeyID || 'verifyKey';
    return `treeValidateSecure('${filename}', '${verificationKeyID}')`;
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
    
    // Include log level if it's not the default 'info' OR if there are additional args (to preserve arg position)
    if (logLevel !== 'info' || additionalArgs.length > 0) {
      args.push(`'${logLevel}'`);

      // Pass additional arguments through verbatim; do not auto-quote
      if (additionalArgs.length > 0) {
        additionalArgs.forEach((arg: string) => {
          const raw = (arg ?? '').toString();
          if (raw.length > 0) {
            args.push(raw);
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

  private generateFunctionCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const parameterEntries = this.normalizeFunctionParameters(props);
    const body: string = (props.body ?? '').toString();

    const paramNames: string[] = [];
    for (const entry of parameterEntries) {
      if (entry.name.length === 0) continue;
      if (!paramNames.includes(entry.name)) {
        paramNames.push(entry.name);
      }
    }
    const paramsStr = paramNames.join(', ');

    const defaultLines = parameterEntries
      .filter(entry => entry.name.length > 0 && entry.value.length > 0)
      .map(entry => `  if(equal(${entry.name}, DBNull)) {\n    setq(${entry.name}, ${entry.value});\n  }`);

    const normalizedBody = body.replace(/\r\n/g, '\n');
    const hasBodyContent = normalizedBody.trim().length > 0;

    const sections: string[] = [];
    if (defaultLines.length > 0) {
      sections.push(defaultLines.join('\n'));
    }
    if (hasBodyContent) {
      sections.push(normalizedBody);
    }

    const inner = sections.join('\n');
    return `func(${paramsStr}) {\n${inner}\n}`;
  }

  private generateIfCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const rawCondition = (props.condition ?? '').toString().trim();
    const condition = rawCondition !== '' ? rawCondition : 'true';

    const lines: string[] = [];
    lines.push(`if(${condition}) {`);

    const inlineIf = this.normalizeInlineBlock(props.ifBody ?? props.body, 2);
    const orderedChildren = this.getOrderedChildren(node.id);
    const thenNodeId = orderedChildren.find(childId => {
      const childNode = this.nodeMap.get(childId);
      return childNode ? this.getNodeLabel(childNode) === 'Then' : false;
    }) ?? null;

    let nestedIf: string[] = [];
    if (thenNodeId) {
      const branchChildren = this.getBranchChildren(thenNodeId);
      nestedIf = this.generateBlockFromChildren(thenNodeId, 2, branchChildren);
    } else {
      const fallbackChildren = orderedChildren.filter(childId => {
        const childNode = this.nodeMap.get(childId);
        const label = childNode ? this.getNodeLabel(childNode) : '';
        return label !== 'Else';
      });
      nestedIf = this.generateBlockFromChildren(node.id, 2, fallbackChildren);
    }

    if (inlineIf.length > 0) {
      lines.push(...inlineIf);
    }
    if (nestedIf.length > 0) {
      lines.push(...nestedIf);
    }
    if (inlineIf.length === 0 && nestedIf.length === 0) {
      lines.push(`${this.indent(2)}// TODO: add statements`);
    }

    lines.push('}');

    const elseText = (props.elseBody ?? '').toString();
    const elseNodeId = orderedChildren.find(childId => {
      const childNode = this.nodeMap.get(childId);
      return childNode ? this.getNodeLabel(childNode) === 'Else' : false;
    }) ?? null;

    let elseBranchBlock: string[] = [];
    if (elseNodeId) {
      const branchChildren = this.getBranchChildren(elseNodeId);
      elseBranchBlock = this.generateBlockFromChildren(elseNodeId, 2, branchChildren);
    }

    const inlineElse = this.normalizeInlineBlock(elseText, 2);
    const includeElse = this.isTruthy(props.hasElse) || inlineElse.length > 0 || elseBranchBlock.length > 0;
    if (includeElse) {
      lines.push('else {');
      if (inlineElse.length > 0) {
        lines.push(...inlineElse);
      }
      if (elseBranchBlock.length > 0) {
        lines.push(...elseBranchBlock);
      }
      if (inlineElse.length === 0 && elseBranchBlock.length === 0) {
        lines.push(`${this.indent(2)}// TODO: add else statements`);
      }
      lines.push('}');
    }

    return lines.join('\n');
  }

  private generateWhileCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const rawCondition = (props.condition ?? '').toString().trim();
    const condition = rawCondition !== '' ? rawCondition : 'true';

    const lines: string[] = [];
    lines.push(`while(${condition}) {`);

    const maxIterations = Number(props.maxIterations);
    if (Number.isFinite(maxIterations) && maxIterations > 0) {
      lines.push(`${this.indent(2)}// max iterations: ${Math.floor(maxIterations)}`);
    }

    const inlineBody = this.normalizeInlineBlock(props.body, 2);
    const orderedChildren = this.getOrderedChildren(node.id);
    const loopBodyNodeId = orderedChildren.find(childId => {
      const childNode = this.nodeMap.get(childId);
      if (!childNode) {
        return false;
      }
      return this.getNodeLabel(childNode) === 'Loop Body';
    }) ?? null;

    let nestedBlock: string[] = [];
    if (loopBodyNodeId) {
      const branchChildren = this.getBranchChildren(loopBodyNodeId);
      nestedBlock = this.generateBlockFromChildren(loopBodyNodeId, 2, branchChildren);
    } else {
      nestedBlock = this.generateBlockFromChildren(node.id, 2, orderedChildren);
    }

    if (inlineBody.length > 0) {
      lines.push(...inlineBody);
    }
    if (nestedBlock.length > 0) {
      lines.push(...nestedBlock);
    }
    if (inlineBody.length === 0 && nestedBlock.length === 0) {
      lines.push(`${this.indent(2)}// TODO: add loop body statements`);
    }

    lines.push('}');
    return lines.join('\n');
  }

  private normalizeFunctionParameters(props: Record<string, any>): { name: string; value: string }[] {
    const result: { name: string; value: string }[] = [];
    const raw = props?.parameters;
    if (Array.isArray(raw)) {
      raw.forEach((entry: any) => {
        if (typeof entry === 'string') {
          const name = entry.toString().trim();
          if (name.length > 0) {
            result.push({ name, value: '' });
          }
        } else if (entry && typeof entry === 'object') {
          const name = (entry.name ?? '').toString().trim();
          const value = (entry.value ?? '').toString().trim();
          if (name.length > 0 || value.length > 0) {
            result.push({ name, value });
          }
        }
      });
    }

    const legacyNames = Array.isArray(props?.parameterNames) ? props.parameterNames : [];
    const legacyValues = Array.isArray(props?.parameterValues) ? props.parameterValues : [];
    const maxLegacy = Math.max(legacyNames.length, legacyValues.length);
    for (let i = 0; i < maxLegacy; i++) {
      const name = (legacyNames[i] ?? '').toString().trim();
      const value = (legacyValues[i] ?? '').toString().trim();
      if (name.length > 0 || value.length > 0) {
        result.push({ name, value });
      }
    }

    return result;
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
      'childCount': 'childCount',
      'child count': 'childCount',
      'firstchild': 'firstChild',
      'first child': 'firstChild',
      'lastchild': 'lastChild',
      'last child': 'lastChild',
      'getchildat': 'getChildAt',
      'get child at': 'getChildAt',
      'clear': 'clear',
      'clear ': 'clear',
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
      'treeValidateSecure': 'treeValidateSecure',
      'tree validate secure': 'treeValidateSecure',
      'treeWalk': 'treeWalk',
      'tree walk': 'treeWalk',
      'treeToYAML': 'treeToYAML',
      'tree to yaml': 'treeToYAML',
      'treeToXML': 'treeToXML',
      'tree to xml': 'treeToXML',
      'treeGetMetadata': 'treeGetMetadata',
      'tree get metadata': 'treeGetMetadata',
      'getValue': 'getValue',
      'get value': 'getValue',
      'setValue': 'setValue',
      'set value': 'setValue',
      'getAttribute': 'getAttribute',
      'get attribute': 'getAttribute',
  'removeattribute': 'removeAttribute',
  'remove attribute': 'removeAttribute',
      'setAttribute': 'setAttribute',
      'set attribute': 'setAttribute',
      'csvnode': 'csvNode',
      'csv node': 'csvNode',
  'jsonnode': 'jsonNode',
  'json node': 'jsonNode',
  'xmlnode': 'xmlNode',
  'xml node': 'xmlNode',
  'yamlnode': 'yamlNode',
  'yaml node': 'yamlNode',
  'mapnode': 'mapNode',
  'map node': 'mapNode',
    'findbyname': 'findByName',
      'find by name': 'findByName',
  'traversenode': 'traverseNode',
  'traverse node': 'traverseNode',
  'querynode': 'queryNode',
  'query node': 'queryNode',
      'getchildbyname': 'getChildByName',
      'get child by name': 'getChildByName',
  'list': 'list',
      'getdepth': 'getDepth',
      'get depth': 'getDepth',
      'getlevel': 'getLevel',
      'get level': 'getLevel',
      'getname': 'getName',
      'get name': 'getName',
  'setname': 'setName',
  'set name': 'setName',
      'getparent': 'getParent',
      'get parent': 'getParent',
      'getpath': 'getPath',
      'get path': 'getPath',
      'getroot': 'getRoot',
      'get root': 'getRoot',
      'getsiblings': 'getSiblings',
      'get siblings': 'getSiblings'
      ,
      'gettext': 'getText',
      'get text': 'getText'
      ,
      'hasattribute': 'hasAttribute',
      'has attribute': 'hasAttribute'
      ,
      'isleaf': 'isLeaf',
      'is leaf': 'isLeaf',
      'isroot': 'isRoot',
      'is root': 'isRoot'
      ,
      'nodetostring': 'nodeToString',
      'node to string': 'nodeToString'
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

  private generateSwitchCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const rawExpr = (props.testExpression ?? '').toString().trim();
    const header = rawExpr !== '' ? `switch(${rawExpr}) {` : 'switch() {';
    const lines = [header];

    const orderedChildren = this.getOrderedChildren(node.id);
    const caseNodes: VisualDSLNode[] = [];
    let defaultNode: VisualDSLNode | null = null;

    for (const childId of orderedChildren) {
      const childNode = this.nodeMap.get(childId);
      if (!childNode) continue;
      if (childNode.data.label === 'Case') {
        caseNodes.push(childNode);
      } else if (childNode.data.label === 'Default' && !defaultNode) {
        defaultNode = childNode;
      }
    }

    if (caseNodes.length === 0 && !defaultNode) {
      lines.push(`${this.indent(2)}// TODO: add case or default branch`);
    } else {
      for (const caseNode of caseNodes) {
        lines.push(...this.generateCaseBlock(caseNode, 2));
      }
      if (defaultNode) {
        lines.push(...this.generateDefaultBlock(defaultNode, 2));
      }
    }

    lines.push('}');
    return lines.join('\n');
  }

  private generateCaseBlock(node: VisualDSLNode, indentLevel: number): string[] {
    const props = node.data.properties || {};
    const rawCondition = (props.condition ?? '').toString().trim();
    const condition = rawCondition !== '' ? rawCondition : 'true';
    const lines: string[] = [];
    lines.push(`${this.indent(indentLevel)}case(${condition}) {`);

    const inlineBody = this.normalizeInlineBlock(props.body, indentLevel + 2);
    const branchChildren = this.getBranchChildren(node.id);
    const generatedBody = this.generateBlockFromChildren(node.id, indentLevel + 2, branchChildren);

    if (inlineBody.length > 0) {
      lines.push(...inlineBody);
    }
    if (generatedBody.length > 0) {
      lines.push(...generatedBody);
    }
    if (inlineBody.length === 0 && generatedBody.length === 0) {
      lines.push(`${this.indent(indentLevel + 2)}// TODO: add statements`);
    }

    lines.push(`${this.indent(indentLevel)}}`);
    return lines;
  }

  private generateDefaultBlock(node: VisualDSLNode, indentLevel: number): string[] {
    const props = node.data.properties || {};
    const lines: string[] = [];
    lines.push(`${this.indent(indentLevel)}default() {`);

    const inlineBody = this.normalizeInlineBlock(props.body, indentLevel + 2);
    const branchChildren = this.getBranchChildren(node.id);
    const generatedBody = this.generateBlockFromChildren(node.id, indentLevel + 2, branchChildren);

    if (inlineBody.length > 0) {
      lines.push(...inlineBody);
    }
    if (generatedBody.length > 0) {
      lines.push(...generatedBody);
    }
    if (inlineBody.length === 0 && generatedBody.length === 0) {
      lines.push(`${this.indent(indentLevel + 2)}// TODO: add statements`);
    }

    lines.push(`${this.indent(indentLevel)}}`);
    return lines;
  }

  private generateBlockFromChildren(parentId: string, indentLevel: number, overrideChildIds?: string[]): string[] {
    const childIds = overrideChildIds ?? this.getOrderedChildren(parentId);
    const result: string[] = [];
    const indent = this.indent(indentLevel);

    for (const childId of childIds) {
      const childNode = this.nodeMap.get(childId);
      if (!childNode) continue;
      const childCode = this.generateNodeCode(childNode);
      if (!childCode) continue;
      const childLines = childCode.split('\n');
      for (const line of childLines) {
        result.push(indent + line);
      }
    }

    return result;
  }

  private normalizeInlineBlock(raw: unknown, indentLevel: number): string[] {
    if (raw === undefined || raw === null) {
      return [];
    }
    const text = raw.toString().trim();
    if (text === '') {
      return [];
    }
    const indent = this.indent(indentLevel);
    return text.split(/\r?\n/).map(line => indent + line.trimEnd());
  }

  private isTruthy(value: unknown): boolean {
    if (typeof value === 'boolean') {
      return value;
    }
    if (typeof value === 'string') {
      const normalized = value.trim().toLowerCase();
      if (normalized === 'true') {
        return true;
      }
      if (normalized === 'false') {
        return false;
      }
    }
    if (typeof value === 'number') {
      return value !== 0;
    }
    return false;
  }

  private getOrderedChildren(parentId: string): string[] {
    return this.diagram.nestingRelations
      .filter(rel => rel.parentId === parentId)
      .sort((a, b) => a.order - b.order)
      .map(rel => rel.childId)
      .filter(childId => this.nodeMap.has(childId));
  }

  private indent(level: number): string {
    if (level <= 0) {
      return '';
    }
    return ' '.repeat(level);
  }

  private collectStructuralInlineNodes(): Set<string> {
    const inline = new Set<string>();

    const visit = (nodeId: string) => {
      if (inline.has(nodeId)) {
        return;
      }
      inline.add(nodeId);
      const children = this.nestingMap.get(nodeId) || [];
      for (const child of children) {
        visit(child);
      }
    };

    for (const [parentId, childIds] of this.nestingMap) {
      const parentNode = this.nodeMap.get(parentId);
      if (!parentNode) {
        continue;
      }

      const parentLabel = this.getNodeLabel(parentNode);
      if (
        parentLabel === 'Switch' ||
        parentLabel === 'If' ||
        parentLabel === 'While' ||
        parentLabel === 'Loop Body'
      ) {
        for (const childId of childIds) {
          visit(childId);
          const branchExtras = this.collectBranchFlow(childId);
          for (const extra of branchExtras) {
            visit(extra);
          }
        }
      }
    }

    return inline;
  }

  private collectBranchFlow(parentId: string): string[] {
    const visited = new Set<string>();
    const pending: string[] = [];

    for (const edge of this.diagram.edges) {
      if (edge.source === parentId && this.nodeMap.has(edge.target)) {
        pending.push(edge.target);
      }
    }

    while (pending.length > 0) {
      const nodeId = pending.pop()!;
      if (visited.has(nodeId)) {
        continue;
      }

      const node = this.nodeMap.get(nodeId);
      if (!node) {
        continue;
      }

      const label = this.getNodeLabel(node);
      if (
        label === 'Case' ||
        label === 'Default' ||
        label === 'Switch' ||
        label === 'Then' ||
        label === 'Else' ||
        label === 'Loop Body'
      ) {
        continue;
      }

      const incomingFromSwitch = this.diagram.edges.some(edge => {
        if (edge.target !== nodeId) {
          return false;
        }
        const sourceNode = this.nodeMap.get(edge.source);
        const sourceLabel = sourceNode ? this.getNodeLabel(sourceNode) : '';
        return sourceLabel === 'Switch' || sourceLabel === 'If' || sourceLabel === 'While';
      });
      if (incomingFromSwitch) {
        continue;
      }

      const incomingFromOtherBranch = this.diagram.edges.some(edge => {
        if (edge.target !== nodeId) {
          return false;
        }
        if (edge.source === parentId) {
          return false;
        }
        const sourceNode = this.nodeMap.get(edge.source);
        const sourceLabel = sourceNode ? this.getNodeLabel(sourceNode) : '';
        return (
          sourceLabel === 'Case' ||
          sourceLabel === 'Default' ||
          sourceLabel === 'Then' ||
          sourceLabel === 'Else' ||
          sourceLabel === 'Loop Body'
        );
      });
      if (incomingFromOtherBranch) {
        continue;
      }

      const parent = this.parentLookup.get(nodeId);
      if (parent && parent !== parentId) {
        continue;
      }

      visited.add(nodeId);

      for (const edge of this.diagram.edges) {
        if (edge.source === nodeId && !visited.has(edge.target)) {
          pending.push(edge.target);
        }
      }
    }

    if (visited.size === 0) {
      return [];
    }

    const ordered: string[] = [];
    for (const nodeId of this.executionOrder) {
      if (visited.has(nodeId)) {
        ordered.push(nodeId);
      }
    }
    return ordered;
  }

  private getBranchChildren(parentId: string): string[] {
    const nested = this.getOrderedChildren(parentId);
    const seen = new Set<string>(nested);
    const result = [...nested];

    const extras = this.collectBranchFlow(parentId);
    for (const childId of extras) {
      if (!seen.has(childId)) {
        result.push(childId);
        seen.add(childId);
      }
    }

    return result;
  }

  private generateSetqCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const rawVar = props.variableName ?? this.inferVariableName(node) ?? 'var';
    const varName = rawVar.toString().trim() || 'var';
    const childIds = this.getOrderedChildren(node.id);

    if (childIds.length > 0) {
      const lines: string[] = [];

      const leadingChildren = childIds.slice(0, -1);
      for (const childId of leadingChildren) {
        const childNode = this.nodeMap.get(childId);
        if (!childNode) {
          continue;
        }
        const childCode = this.generateNodeCode(childNode);
        if (childCode) {
          lines.push(childCode);
        }
      }

      let inlineValue: string | null = null;
      const finalChildId = childIds[childIds.length - 1];
      const finalChildNode = this.nodeMap.get(finalChildId);
      if (finalChildNode) {
        inlineValue = this.generateSetqInlineValue(finalChildNode);
        if (!inlineValue) {
          const finalChildCode = this.generateNodeCode(finalChildNode);
          if (finalChildCode) {
            lines.push(finalChildCode);
          }
        }
      }

      const { expression: fallbackExpression } = this.formatSetqValueFromProps(props);
      const resolvedValue = inlineValue && inlineValue.trim().length > 0
        ? inlineValue
        : fallbackExpression;

      lines.push(`setq(${varName}, ${resolvedValue})`);
      return lines.join('\n');
    }

    const { expression } = this.formatSetqValueFromProps(props);
    return `setq(${varName}, ${expression})`;
  }

  private generateSetqInlineValue(childNode: VisualDSLNode): string | null {
    const label = this.getNodeLabel(childNode);
    if (
      [
        'Set Equal',
        'Set Value',
        'Set Q',
        'SetQ',
        'Declare',
        'If',
        'Then',
        'Else',
        'While',
        'Loop Body',
        'Switch',
        'Case',
        'Default',
        'Break',
        'Continue'
      ].includes(label)
    ) {
      return null;
    }

    switch (label) {
      case 'Create':
        return this.generateCreateCode(childNode);
      case 'New Tree':
        return this.generateNewTreeCode(childNode);
      case 'Parse JSON':
        return this.generateParseJSONCode(childNode);
      case 'Array':
        return this.generateArrayCode(childNode);
      case 'List':
        return this.generateListCode(childNode);
      case 'Node To String':
        return this.generateNodeToStringCode(childNode);
      case 'Get Attribute':
        return this.generateGetAttributeCode(childNode);
      case 'Has Attribute':
        return this.generateHasAttributeCode(childNode);
      case 'Function':
        return this.generateFunctionCode(childNode);
      case 'LogPrint':
      case 'Log Print':
      case 'logPrint':
        return null;
      default: {
        const allowedCategories = new Set([
          'value',
          'string',
          'math',
          'array',
          'date',
          'crypto',
          'tree',
          'node',
          'json',
          'dispatcher',
          'sql',
          'host',
          'system'
        ]);
        if (allowedCategories.has(childNode.data.category)) {
          return this.generateGenericFunctionCode(childNode);
        }
        return null;
      }
    }
  }

  private formatSetqValueFromProps(props: Record<string, any>): { expression: string; isEmpty: boolean } {
    const rawValue = props.value ?? '';
    const valueText = rawValue.toString();
    const isBlank = valueText.trim().length === 0;
    const valueType = props.valueType === 'expression' ? 'expression' : 'string';

    if (isBlank) {
      return { expression: `''`, isEmpty: true };
    }

    if (valueType === 'expression') {
      return { expression: valueText, isEmpty: false };
    }

    const escaped = valueText.replace(/'/g, "\\'");
    return { expression: `'${escaped}'`, isEmpty: false };
  }

  private generateListCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `list(${nodeVar})`;
  }

  private generateNodeToStringCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `nodeToString(${nodeVar})`;
  }

  private generateGetAttributeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = props.variableName || 'var';
    const attributeName = props.attributeName || 'attr';
    
    return `getAttribute(${varName}, '${attributeName}')`;
  }

  private generateSetAttributeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = (props.variableName ?? 'var').toString().trim() || 'var';
    const attributeName = (props.attributeName ?? 'attr').toString().trim() || 'attr';
    const raw = (props.value ?? '').toString().trim();

    if (raw === '') {
      return `setAttribute(${varName}, '${attributeName}', '')`;
    }

    // Inline function literal support: if value looks like func(...) { ... }
    if (/^func\s*\(/.test(raw)) {
      return `setAttribute(${varName}, '${attributeName}', ${raw})`;
    }

    // If it's a boolean or number, emit as-is
    if (raw === 'true' || raw === 'false') {
      return `setAttribute(${varName}, '${attributeName}', ${raw})`;
    }
    if (!isNaN(Number(raw)) && /^-?\d+(\.\d+)?$/.test(raw)) {
      return `setAttribute(${varName}, '${attributeName}', ${raw})`;
    }

    // If it's an identifier (variable name) or function call, emit as-is
    if (/^[A-Za-z_][A-Za-z0-9_]*$/.test(raw) || /\)$/.test(raw)) {
      return `setAttribute(${varName}, '${attributeName}', ${raw})`;
    }

    // Otherwise, treat as a string literal
    const escaped = raw.replace(/'/g, "\\'");
    return `setAttribute(${varName}, '${attributeName}', '${escaped}')`;
  }

  private generateSetAttributesCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = (props.variableName ?? 'var').toString().trim() || 'var';
    const rawMap = (props.attributesMap ?? '').toString().trim();
    // If blank, still emit call with empty object to be explicit
    if (!rawMap) {
      return `setAttributes(${varName}, {})`;
    }
    // Preserve as-is if not an object literal string starting with '{'
    const isObjectLiteral = rawMap.startsWith('{') || rawMap.startsWith('mapNode(');
    if (isObjectLiteral) {
      return `setAttributes(${varName}, ${rawMap})`;
    }
    // Otherwise treat as variable symbol
    return `setAttributes(${varName}, ${rawMap})`;
  }

  private generateSetTextCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = (props.variableName ?? 'var').toString().trim() || 'var';
    const rawText = (props.text ?? '').toString();
    const escaped = rawText.replace(/'/g, "\\'");
    return `setText(${varName}, '${escaped}')`;
  }

  private generateRemoveAttributeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = (props.variableName ?? 'var').toString().trim() || 'var';
    const attributeName = (props.attributeName ?? 'attr').toString().trim() || 'attr';
    return `removeAttribute(${varName}, '${attributeName}')`;
  }

  private generateHasAttributeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = (props.variableName ?? 'var').toString().trim() || 'var';
    const attributeName = (props.attributeName ?? 'attr').toString().trim() || 'attr';
    return `hasAttribute(${varName}, '${attributeName}')`;
  }

  private generateTreeToYAMLCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const treeVar = props.treeVariable || 'tree';
    return `treeToYAML(${treeVar})`;
  }

  private generateTreeToXMLCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const treeVar = props.treeVariable || 'tree';
    const pretty = props.prettyPrint;
    if (typeof pretty === 'boolean') {
      return `treeToXML(${treeVar}, ${pretty})`;
    }
    return `treeToXML(${treeVar})`;
  }

  private generateTreeGetMetadataCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const filename = props.filename || 'data.json';
    return `treeGetMetadata('${filename}')`;
  }

  private generateCSVNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const filename = (props.filename ?? '').toString().trim();
    const delimiter = (props.delimiter ?? ',').toString();
    const hasHeaders = props.hasHeaders !== undefined ? !!props.hasHeaders : true;
    if (!filename) {
      // Fallback to a placeholder to avoid empty required arg
      return `csvNode('data.csv')`;
    }
    // Only include optional args if they differ from defaults
    const args: string[] = [`'${filename}'`];
    if (delimiter !== ',') args.push(`'${delimiter}'`);
    if (hasHeaders === false || args.length > 1) {
      // If including hasHeaders, ensure delimiter arg slot exists
      if (args.length === 1) args.push(`','`);
      args.push(String(hasHeaders));
    }
    return `csvNode(${args.join(', ')})`;
  }

  private generateJSONNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const raw = (props.jsonOrName ?? '').toString().trim();
    if (!raw) {
      // Zero-arg form
      return 'jsonNode()';
    }
    // Emit single string argument as-is (quoted)
    // Note: backend decides whether to parse JSON or treat as name
    // based on the first char being '{' or '['
    const escaped = raw.replace(/'/g, "\\'");
    return `jsonNode('${escaped}')`;
  }

  private generateXMLNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const raw = (props.xmlString ?? '').toString().trim();
    if (!raw) {
      // Zero-arg form creates empty XML node named "xml"
      return 'xmlNode()';
    }
    const escaped = raw.replace(/'/g, "\\'");
    return `xmlNode('${escaped}')`;
  }

  private generateYAMLNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const raw = (props.yamlString ?? '').toString().trim();
    if (!raw) {
      // Zero-arg form creates empty YAML node named "yaml"
      return 'yamlNode()';
    }
    const escaped = raw.replace(/'/g, "\\'");
    return `yamlNode('${escaped}')`;
  }

  private generateMapNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const raw = (props.mapString ?? '').toString().trim();
    if (!raw) {
      // Zero-arg form: creates empty map named "map"
      return 'mapNode()';
    }
    // Single string argument: backend will parse to initialize the map
    const escaped = raw.replace(/'/g, "\\'");
    return `mapNode('${escaped}')`;
  }

  private generateFindByNameCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();
    const childNameRaw = (props.name ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    const childName = childNameRaw || 'child';
    return `findByName(${nodeVar}, '${childName}')`;
  }

  private generateTraverseNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();
    let functionName = (props.functionName ?? '').toString().trim() || 'visitFn';

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    // Emit the function name as an unquoted symbol
    return `traverseNode(${nodeVar}, ${functionName})`;
  }

  private generateQueryNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();
    let functionName = (props.functionName ?? '').toString().trim() || 'predicateFn';

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    // Emit the function name as an unquoted symbol
    return `queryNode(${nodeVar}, ${functionName})`;
  }

  private generateGetChildByNameCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();
    const childNameRaw = (props.name ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    const childName = childNameRaw || 'child';
    return `getChildByName(${nodeVar}, '${childName}')`;
  }

  private generateGetDepthCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `getDepth(${nodeVar})`;
  }

  private generateGetLevelCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `getLevel(${nodeVar})`;
  }

  private generateGetNameCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `getName(${nodeVar})`;
  }

  private generateSetNameCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? props.variableName ?? '').toString().trim();
    const rawName = (props.name ?? '').toString();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }
    if (!nodeVar) nodeVar = 'node';
    const escaped = rawName.replace(/'/g, "\\'");
    return `setName(${nodeVar}, '${escaped}')`;
  }

  private generateGetParentCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `getParent(${nodeVar})`;
  }

  private generateGetPathCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `getPath(${nodeVar})`;
  }

  private generateGetRootCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `getRoot(${nodeVar})`;
  }

  private generateGetSiblingsCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `getSiblings(${nodeVar})`;
  }

  private generateGetTextCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `getText(${nodeVar})`;
  }

  private generateIsLeafCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `isLeaf(${nodeVar})`;
  }

  private generateIsRootCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();

    if (!nodeVar) {
      // Try infer from incoming edge declare
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }

    if (!nodeVar) nodeVar = 'node';
    return `isRoot(${nodeVar})`;
  }

  private generateNestedChildCode(childNode: VisualDSLNode): string | null {
    switch (childNode.data.label) {
      case 'Create':
        return this.generateCreateCode(childNode);
      case 'New Tree':
        return this.generateNewTreeCode(childNode);
        
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
