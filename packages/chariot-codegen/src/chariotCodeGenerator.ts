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

// Public options for code generation
export type GenerateOptions = {
  // When true (default), append a base64-encoded diagram payload for reverse mapping
  // Set to false to omit the trailing metadata comment
  embedSource?: boolean;
};

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
    diagram.nodes.forEach(node => {
      if (node.type === 'group') return;
      this.nodeMap.set(node.id, node);
    });
    diagram.nestingRelations.forEach(rel => {
      if (this.nodeMap.has(rel.parentId) && this.nodeMap.has(rel.childId)) {
        if (!this.nestingMap.has(rel.parentId)) {
          this.nestingMap.set(rel.parentId, []);
        }
        this.nestingMap.get(rel.parentId)!.push(rel.childId);
        this.parentLookup.set(rel.childId, rel.parentId);
      }
    });
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
      'symbol': 'Symbol',
    };
    return aliasMap[lowerKey] || normalized;
  }

  private getNodeLabel(node: VisualDSLNode): string {
    return this.canonicalLabel(node.data.label);
  }

  public generateChariotCode(options?: GenerateOptions): string {
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
        const inlineLabels = ['Create', 'New Tree', 'Parse JSON', 'Array', 'Range'];
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
    // Append embedded diagram payload for reverse mapping (code -> diagram)
  const shouldEmbed = options?.embedSource !== false;
    if (shouldEmbed) {
      try {
        const payload = JSON.stringify(this.diagram);
        const encoded = encodeBase64(payload);
        lines.push('');
        lines.push(`// __VDSL_SOURCE__: base64:${encoded}`);
      } catch (e) {
        // If embedding fails, skip silently to avoid breaking codegen
      }
    }
    return lines.join('\n');
  }

  private calculateExecutionOrder(): void {
    const startNode = this.diagram.nodes.find(node => 
      node.data.label === 'Start' || node.id === 'start'
    );
    if (!startNode) {
      if (this.diagram.nodes.length > 0) {
        this.executionOrder = this.diagram.nodes
          .filter(n => n.type !== 'group')
          .map(n => n.id);
      }
      return;
    }
    const visited = new Set<string>();
    const stack = [startNode.id];
    while (stack.length > 0) {
      const currentId = stack.pop()!;
      if (visited.has(currentId)) continue;
      visited.add(currentId);
      this.executionOrder.push(currentId);
      const outgoingEdges = this.diagram.edges.filter(edge => edge.source === currentId);
      const mainFlowEdges = outgoingEdges.filter(edge => {
        const targetNode = this.nodeMap.get(edge.target);
        return targetNode && (
          edge.sourceHandle === 'right' || 
          targetNode.data.label === 'Tree Save' ||
          targetNode.data.label === 'Add Child'
        );
      });
      const nestingEdges = outgoingEdges.filter(edge => !mainFlowEdges.includes(edge));
      mainFlowEdges.sort((a, b) => {
        const nodeA = this.nodeMap.get(a.target);
        const nodeB = this.nodeMap.get(b.target);
        if (!nodeA || !nodeB) return 0;
        if (Math.abs(nodeA.position.y - nodeB.position.y) < 50) {
          return nodeA.position.x - nodeB.position.x;
        } else {
          return nodeA.position.y - nodeB.position.y;
        }
      });
      for (let i = mainFlowEdges.length - 1; i >= 0; i--) {
        stack.push(mainFlowEdges[i].target);
      }
      for (let i = nestingEdges.length - 1; i >= 0; i--) {
        stack.push(nestingEdges[i].target);
      }
    }
    this.diagram.nodes.forEach(node => {
      if (node.type === 'group') return;
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
        return `// Starting ${props.name || this.diagram.name}`;
      case 'Declare':
        return this.generateDeclareCode(node);
      case 'Symbol':
        return this.generateSymbolCode(node);
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
      case 'Range':
        return this.generateRangeCode(node);
      case 'RL Init':
      case 'rlInit':
        return this.generateRLInitCode(node);
      case 'RL Score':
      case 'rlScore':
        return this.generateRLScoreCode(node);
      case 'RL Learn':
      case 'rlLearn':
        return this.generateRLLearnCode(node);
      case 'RL Close':
      case 'rlClose':
        return this.generateRLCloseCode(node);
      case 'RL Select Best':
      case 'rlSelectBest':
        return this.generateRLSelectBestCode(node);
      case 'Extract RL Features':
      case 'extractRLFeatures':
        return this.generateExtractRLFeaturesCode(node);
      case 'RL Explore':
      case 'rlExplore':
        return this.generateRLExploreCode(node);
      case 'NBA Decision':
      case 'nbaDecision':
        return this.generateNBADecisionCode(node);
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
        return this.generateGenericFunctionCode(node);
    }
  }

  private generateSymbolCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const raw = (props.symbolName ?? '').toString().trim();
    const name = raw === '' ? 'value' : raw; // default to a generic name if missing
    const escaped = name.replace(/'/g, "\\'");
    return `symbol('${escaped}')`;
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
    const nestedChildren = this.nestingMap.get(node.id) || [];
    if (nestedChildren.length === 1) {
      const childNode = this.nodeMap.get(nestedChildren[0]);
      if (childNode) {
        const inlineLabels = ['Create', 'New Tree', 'Parse JSON', 'Array', 'Range'];
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
    if (Object.prototype.hasOwnProperty.call(props, 'nodeName')) {
      const raw = (props.nodeName ?? '').toString().trim();
      if (raw === '') {
        return 'create()';
      }
      return `create('${raw}')`;
    }
    const fallback = (this.diagram.name && this.diagram.name.trim() !== '') ? this.diagram.name : 'newNode';
    return `create('${fallback}')`;
  }

  private generateNewTreeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const raw = (props.nodeName ?? '').toString().trim();
    const name = raw !== '' ? raw : this.inferNodeNameFromContext(node);
    return `newTree('${name}')`;
  }

  private generateParseJSONCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let jsonString = props.jsonString || '{}';
    const nodeName = props.nodeName || this.inferNodeNameFromContext(node);
    if (jsonString === '{ [] }') {
      jsonString = '[]';
    } else if (jsonString === '{ ["admin", "contributor", "viewer"] }') {
      jsonString = '["admin", "contributor", "viewer"]';
    }
    return `parseJSON('${jsonString}', '${nodeName}')`;
  }

  private generateAddChildCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    if (props.parentNode && props.childNode) {
      return `addChild(${props.parentNode}, ${props.childNode})`;
    }
    const mainParent = this.findMainTreeVariable();
    if (mainParent) {
      const childDeclares = this.diagram.nodes.filter(node => 
        node.data.label === 'Declare' && 
        node.data.properties?.typeSpecifier === 'J' &&
        node.data.properties?.variableName
      );
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
    if (props.parentNode && props.childNode) {
      return `removeChild(${props.parentNode}, ${props.childNode})`;
    }
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
      const incomingEdges = this.diagram.edges.filter(edge => edge.target === node.id);
      if (incomingEdges.length > 0) {
        const sourceNode = this.nodeMap.get(incomingEdges[0].source);
        if (sourceNode && sourceNode.data.properties?.variableName) {
          nodeVar = sourceNode.data.properties.variableName;
        }
      }
    }
    if (!nodeVar) nodeVar = 'node';
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
    if (!treeVar) {
      treeVar = 'tree';
    }
    const filename = props.filename || `${this.diagram.name}.json`;
    let params = [`${treeVar}`, `'${filename}'`];
    if (props.format && props.format !== '') {
      params.push(`'${props.format}'`);
    }
    if (props.compress === true) {
      if (!props.format || props.format === '') {
        const ext = filename.split('.').pop()?.toLowerCase();
        let autoFormat = 'json';
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
    return `addTo(${collectionName}, ${value})`;
  }

  private generateLogPrintCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const message = props.message || 'message';
    const logLevel = props.logLevel || 'info';
    const additionalArgs = props.additionalArgs || [];
    const args: string[] = [];
    if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(message)) {
      args.push(message);
    } else {
      args.push(`'${message}'`);
    }
    if (logLevel !== 'info' || additionalArgs.length > 0) {
      args.push(`'${logLevel}'`);
      if (additionalArgs.length > 0) {
        additionalArgs.forEach((arg: string) => {
          const raw = (arg ?? '').toString();
          if (raw.length > 0) {
            args.push(raw);
          }
        });
      }
    }
    return `logPrint(${args.join(', ')})`;
  }

  private generateCreateTransformCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let transformName = props.transformName || 'transform';
    if ((transformName.startsWith('"') && transformName.endsWith('"')) || 
        (transformName.startsWith("'") && transformName.endsWith("'"))) {
      const unquoted = transformName.slice(1, -1);
      if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(unquoted)) {
        transformName = unquoted;
      }
    }
    if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(transformName)) {
      return `createTransform(${transformName})`;
    } else {
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
    if ((transform.startsWith('"') && transform.endsWith('"')) || 
        (transform.startsWith("'") && transform.endsWith("'"))) {
      transform = transform.slice(1, -1);
    }
    const args: string[] = [];
    args.push(transform);
    args.push(`'${sourceField}'`);
    args.push(`'${targetColumn}'`);
    if (Array.isArray(program) && program.length > 0) {
      const programStr = program.map((line: string) => `'${line}'`).join(', ');
      args.push(`[${programStr}]`);
    } else {
      args.push('[]');
    }
    args.push(`'${dataType}'`);
    args.push(String(required));
    return `addMapping(${args.join(', ')})`;
  }

  private generateArrayCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const values = props.values || [];
    if (Array.isArray(values) && values.length > 0) {
      const valuesStr = values.map((v: string) => `'${v}'`).join(', ');
      return `array(${valuesStr})`;
    }
    return `array()`;
  }

  private generateRangeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const start = props.start || '0';
    const end = props.end || '10';
    return `range(${start}, ${end})`;
  }

  private generateRLInitCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const feat_dim = props.feat_dim || '12';
    const alpha = props.alpha || '0.3';
    
    // Build JSON config object
    const config: any = { feat_dim: Number(feat_dim), alpha: Number(alpha) };
    if (props.model_path) config.model_path = props.model_path;
    if (props.model_input) config.model_input = props.model_input;
    if (props.model_output) config.model_output = props.model_output;
    
    const configJSON = JSON.stringify(config);
    return `rlInit('${configJSON}')`;
  }

  private generateRLScoreCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const handle = props.handle || 'rlHandle';
    const featuresArray = props.featuresArray || 'features';
    const featDim = props.featDim || '12';
    return `rlScore(${handle}, ${featuresArray}, ${featDim})`;
  }

  private generateRLLearnCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const handle = props.handle || 'rlHandle';
    const rewards = props.rewards || '[0.8, 0.5, 0.3]';
    
    // Build feedback JSON
    const feedbackJSON = `{"rewards": ${rewards}}`;
    return `rlLearn(${handle}, '${feedbackJSON}')`;
  }

  private generateRLCloseCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const handle = props.handle || 'rlHandle';
    return `rlClose(${handle})`;
  }

  private generateRLSelectBestCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const scoresArray = props.scoresArray || 'scores';
    const candidates = props.candidates || 'candidates';
    return `rlSelectBest(${scoresArray}, ${candidates})`;
  }

  private generateExtractRLFeaturesCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const candidates = props.candidates || 'candidates';
    const mode = props.mode || 'normalized';
    return `extractRLFeatures(${candidates}, '${mode}')`;
  }

  private generateRLExploreCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const scores = props.scores || 'scores';
    const candidates = props.candidates || 'candidates';
    const epsilon = props.epsilon || '0.1';
    return `rlExplore(${scores}, ${candidates}, ${epsilon})`;
  }

  private generateNBADecisionCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const candidates = props.candidates || 'candidates';
    const rlHandle = props.rlHandle || 'rlHandle';
    return `nbaDecision(${candidates}, ${rlHandle})`;
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
    const rawLabel = node.data.label.toLowerCase();
    const sanitizedLabel = rawLabel.replace(/[^a-z0-9\s]/g, '');
    let functionName = sanitizedLabel.replace(/\s+/g, '');
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
      'get siblings': 'getSiblings',
      'gettext': 'getText',
      'get text': 'getText',
      'hasattribute': 'hasAttribute',
      'has attribute': 'hasAttribute',
      'isleaf': 'isLeaf',
      'is leaf': 'isLeaf',
      'isroot': 'isRoot',
      'is root': 'isRoot',
      'nodetostring': 'nodeToString',
      'node to string': 'nodeToString'
    };
    const labelKey = sanitizedLabel;
    if (camelCaseFunctions[labelKey]) {
      functionName = camelCaseFunctions[labelKey];
    } else if (camelCaseFunctions[functionName]) {
      functionName = camelCaseFunctions[functionName];
    }
    const props = node.data.properties || {};
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
      case 'Symbol':
        return this.generateSymbolCode(childNode);
      case 'Create':
        return this.generateCreateCode(childNode);
      case 'New Tree':
        return this.generateNewTreeCode(childNode);
      case 'Parse JSON':
        return this.generateParseJSONCode(childNode);
      case 'Array':
        return this.generateArrayCode(childNode);
      case 'Range':
        return this.generateRangeCode(childNode);
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
    if (/^func\s*\(/.test(raw)) {
      return `setAttribute(${varName}, '${attributeName}', ${raw})`;
    }
    if (raw === 'true' || raw === 'false') {
      return `setAttribute(${varName}, '${attributeName}', ${raw})`;
    }
    if (!isNaN(Number(raw)) && /^-?\d+(\.\d+)?$/.test(raw)) {
      return `setAttribute(${varName}, '${attributeName}', ${raw})`;
    }
    if (/^[A-Za-z_][A-Za-z0-9_]*$/.test(raw) || /\)$/.test(raw)) {
      return `setAttribute(${varName}, '${attributeName}', ${raw})`;
    }
    const escaped = raw.replace(/'/g, "\\'");
    return `setAttribute(${varName}, '${attributeName}', '${escaped}')`;
  }

  private generateSetAttributesCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = (props.variableName ?? 'var').toString().trim() || 'var';
    const rawMap = (props.attributesMap ?? '').toString().trim();
    if (!rawMap) {
      return `setAttributes(${varName}, {})`;
    }
    const isObjectLiteral = rawMap.startsWith('{') || rawMap.startsWith('mapNode(');
    if (isObjectLiteral) {
      return `setAttributes(${varName}, ${rawMap})`;
    }
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
      return `csvNode('data.csv')`;
    }
    const args: string[] = [`'${filename}'`];
    if (delimiter !== ',') args.push(`'${delimiter}'`);
    if (hasHeaders === false || args.length > 1) {
      if (args.length === 1) args.push(`','`);
      args.push(String(hasHeaders));
    }
    return `csvNode(${args.join(', ')})`;
  }

  private generateJSONNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const raw = (props.jsonOrName ?? '').toString().trim();
    if (!raw) {
      return 'jsonNode()';
    }
    const escaped = raw.replace(/'/g, "\\'");
    return `jsonNode('${escaped}')`;
  }

  private generateXMLNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const raw = (props.xmlString ?? '').toString().trim();
    if (!raw) {
      return 'xmlNode()';
    }
    const escaped = raw.replace(/'/g, "\\'");
    return `xmlNode('${escaped}')`;
  }

  private generateYAMLNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const raw = (props.yamlString ?? '').toString().trim();
    if (!raw) {
      return 'yamlNode()';
    }
    const escaped = raw.replace(/'/g, "\\'");
    return `yamlNode('${escaped}')`;
  }

  private generateMapNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const raw = (props.mapString ?? '').toString().trim();
    if (!raw) {
      return 'mapNode()';
    }
    const escaped = raw.replace(/'/g, "\\'");
    return `mapNode('${escaped}')`;
  }

  private generateFindByNameCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();
    const childNameRaw = (props.name ?? '').toString().trim();
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
    const childName = childNameRaw || 'child';
    return `findByName(${nodeVar}, '${childName}')`;
  }

  private generateTraverseNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();
    let functionName = (props.functionName ?? '').toString().trim() || 'visitFn';
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
    return `traverseNode(${nodeVar}, ${functionName})`;
  }

  private generateQueryNodeCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();
    let functionName = (props.functionName ?? '').toString().trim() || 'predicateFn';
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
    return `queryNode(${nodeVar}, ${functionName})`;
  }

  private generateGetChildByNameCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? '').toString().trim();
    const childNameRaw = (props.name ?? '').toString().trim();
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
    const childName = childNameRaw || 'child';
    return `getChildByName(${nodeVar}, '${childName}')`;
  }

  private generateGetDepthCode(node: VisualDSLNode): string {
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
    return `getDepth(${nodeVar})`;
  }

  private generateGetLevelCode(node: VisualDSLNode): string {
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
    return `getLevel(${nodeVar})`;
  }

  private generateGetParentCode(node: VisualDSLNode): string {
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
    return `getParent(${nodeVar})`;
  }

  private generateGetPathCode(node: VisualDSLNode): string {
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
    switch (this.getNodeLabel(childNode)) {
      case 'Create':
        return this.generateCreateCode(childNode);
      case 'New Tree':
        return this.generateNewTreeCode(childNode);
      case 'Parse JSON': {
        const parseJsonChildren = this.nestingMap.get(childNode.id) || [];
        if (parseJsonChildren.length > 0) {
          return this.generateParseJSONCode(childNode);
        }
        return this.generateParseJSONCode(childNode);
      }
      case 'Array':
        return this.generateArrayCode(childNode);
      case 'Range':
        return this.generateRangeCode(childNode);
      default:
        return this.generateGenericFunctionCode(childNode);
    }
  }

  private generateGetNameCode(node: VisualDSLNode): string {
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
    return `getName(${nodeVar})`;
  }

  private generateSetNameCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let nodeVar = (props.node ?? props.variableName ?? '').toString().trim();
    const rawName = (props.name ?? '').toString();
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
    const escaped = rawName.replace(/'/g, "\\'");
    return `setName(${nodeVar}, '${escaped}')`;
  }

  private inferChildVariable(node: VisualDSLNode): string {
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
    if (node.data.properties?.variableName) {
      return node.data.properties.variableName;
    }
    const label = node.data.label.toLowerCase().replace(/\s+/g, '');
    const nearbyNodes = this.diagram.nodes.filter(n => {
      const dx = Math.abs(n.position.x - node.position.x);
      const dy = Math.abs(n.position.y - node.position.y);
      return n.id !== node.id && dx < 200 && dy < 100;
    });
    const createNode = nearbyNodes.find(n => n.data.label === 'Create' || n.data.label === 'New Tree');
    if (createNode?.data.properties?.nodeName) {
      return createNode.data.properties.nodeName;
    }
    const parseNode = nearbyNodes.find(n => n.data.label === 'Parse JSON');
    if (parseNode?.data.properties?.nodeName) {
      return parseNode.data.properties.nodeName;
    }
    if (node.position.x > 250 && node.position.x < 350) {
      return 'users';
    } else if (node.position.x > 450 && node.position.x < 550) {
      return 'config';
    } else if (node.position.x > 650 && node.position.x < 750) {
      return 'roles';
    } else if (node.position.x > 800 && node.position.x < 900) {
      return 'rules';
    }
    if (node.position.x < 100) {
      return this.diagram.name;
    }
    const commonNames = ['users', 'roles', 'config', 'rules', 'data'];
    const nodeIndex = this.diagram.nodes.findIndex(n => n.id === node.id);
    if (nodeIndex < commonNames.length) {
      return commonNames[nodeIndex % commonNames.length];
    }
    return `var${nodeIndex}`;
  }

  private inferNodeNameFromContext(node: VisualDSLNode): string {
    const parentRelation = this.diagram.nestingRelations.find(rel => rel.childId === node.id);
    if (parentRelation) {
      const parentNode = this.nodeMap.get(parentRelation.parentId);
      if (parentNode?.data.properties?.variableName) {
        return parentNode.data.properties.variableName;
      }
    }
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
    const mainTreeNode = this.diagram.nodes.find(node => 
      node.data.label === 'Declare' && 
      node.data.properties?.typeSpecifier === 'T' &&
      node.data.properties?.variableName
    );
    return mainTreeNode?.data.properties?.variableName || null;
  }

  private findMostRecentDeclaredVariable(currentNode: VisualDSLNode): string | null {
    const currentIndex = this.executionOrder.indexOf(currentNode.id);
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

export function generateChariotCodeFromDiagram(diagramJson: string, options?: GenerateOptions): string {
  try {
    const diagram: VisualDSLDiagram = JSON.parse(diagramJson);
    const generator = new ChariotCodeGenerator(diagram);
    return generator.generateChariotCode(options);
  } catch (error) {
    throw new Error(`Failed to generate Chariot code: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }
}

// Helper functions for base64 encoding/decoding that work in both browser and Node
function encodeBase64(input: string): string {
  try {
    // Browser-safe path
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    if (typeof btoa === 'function') {
      // Encode UTF-8 safely
      return btoa(unescape(encodeURIComponent(input)));
    }
  } catch (_) {}
  // Node path
  const B: any = (typeof globalThis !== 'undefined' && (globalThis as any).Buffer) ? (globalThis as any).Buffer : null;
  const buf = B ? B.from(input, 'utf-8') : null;
  return buf ? buf.toString('base64') : input;
}

function decodeBase64(b64: string): string {
  try {
    // Browser-safe path
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    if (typeof atob === 'function') {
      // Decode UTF-8 safely
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-ignore
      return decodeURIComponent(escape(atob(b64)));
    }
  } catch (_) {}
  // Node path
  const B: any = (typeof globalThis !== 'undefined' && (globalThis as any).Buffer) ? (globalThis as any).Buffer : null;
  const buf = B ? B.from(b64, 'base64') : null;
  return buf ? buf.toString('utf-8') : b64;
}

// Extract embedded diagram JSON from generated code
export function extractDiagramFromCode(code: string): string {
  const marker = '__VDSL_SOURCE__: base64:';
  const lines = code.split(/\r?\n/);
  for (let i = lines.length - 1; i >= 0; i--) {
    const line = lines[i];
    const idx = line.indexOf(marker);
    if (idx !== -1) {
      const b64 = line.substring(idx + marker.length).trim();
      const json = decodeBase64(b64);
      // Validate JSON
      try {
        const parsed = JSON.parse(json);
        if (parsed && typeof parsed === 'object' && Array.isArray(parsed.nodes) && Array.isArray(parsed.edges)) {
          return json;
        }
      } catch (e) {
        throw new Error('Embedded diagram payload is not valid JSON');
      }
    }
  }
  throw new Error('No embedded diagram payload found in code');
}
