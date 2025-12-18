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
      'and': 'and',
      'or': 'or',
      'not': 'not',
      'equal': 'equal',
      'equals': 'equal',
      'unequal': 'unequal',
      'bigger': 'bigger',
      'greater': 'bigger',
      'smaller': 'smaller',
      'less': 'smaller',
      'biggereq': 'biggerEq',
      'greaterorequal': 'biggerEq',
      'smallereq': 'smallerEq',
      'lessorequal': 'smallerEq',
      'add': 'add',
      'addition': 'add',
      'sub': 'sub',
      'subtract': 'sub',
      'mul': 'mul',
      'multiply': 'mul',
      'div': 'div',
      'divide': 'div',
      'abs': 'abs',
      'absolute': 'abs',
      'max': 'max',
      'maximum': 'max',
      'min': 'min',
      'minimum': 'min',
      'round': 'round',
      'random': 'random',
      'concat': 'concat',
      'concatenate': 'concat',
      'split': 'split',
      'replace': 'replace',
      'substring': 'substring',
      'substr': 'substring',
      'string length': 'strlen',
      'stringlength': 'strlen',
      'strlen': 'strlen',
      'upper': 'upper',
      'uppercase': 'upper',
      'lower': 'lower',
      'lowercase': 'lower',
      'date': 'date',
      'now': 'now',
      'today': 'today',
      'date add': 'dateAdd',
      'dateadd': 'dateAdd',
      'format date': 'formatDate',
      'formatdate': 'formatDate',
      'encrypt': 'encrypt',
      'decrypt': 'decrypt',
      'hash 256': 'hash256',
      'hash256': 'hash256',
      'hash-256': 'hash256',
      'sign': 'sign',
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
        const inlineLabels = ['Create', 'New Tree', 'Parse JSON', 'parseJSON', 'parseJSONSimple', 'Array', 'Range'];
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
      case 'parseJSON':
        return this.generateParseJSONCode(node);
      case 'parseJSONSimple':
      case 'Parse JSON Simple':
        return this.generateParseJSONSimpleCode(node);
      case 'toJSON':
      case 'To JSON':
        return this.generateToJSONCode(node);
      case 'toSimpleJSON':
      case 'To Simple JSON':
        return this.generateToSimpleJSONCode(node);
      case 'csvHeaders':
      case 'CSV Headers':
        return this.generateCSVHeadersCode(node);
      case 'csvRowCount':
      case 'CSV Row Count':
        return this.generateCSVRowCountCode(node);
      case 'csvColumnCount':
      case 'CSV Column Count':
        return this.generateCSVColumnCountCode(node);
      case 'csvGetRow':
      case 'CSV Get Row':
        return this.generateCSVGetRowCode(node);
      case 'csvGetCell':
      case 'CSV Get Cell':
        return this.generateCSVGetCellCode(node);
      case 'csvGetRows':
      case 'CSV Get Rows':
        return this.generateCSVGetRowsCode(node);
      case 'csvToCSV':
      case 'CSV to CSV':
        return this.generateCSVToCSVCode(node);
      case 'csvLoad':
      case 'CSV Load':
      case 'CSV node load from file':
        return this.generateCSVLoadCode(node);
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
      case 'concat':
        return this.generateConcatCode(node);
      case 'split':
        return this.generateSplitCode(node);
      case 'replace':
        return this.generateReplaceCode(node);
      case 'substring':
        return this.generateSubstringCode(node);
      case 'strlen':
        return this.generateStringLengthCode(node);
      case 'upper':
        return this.generateUpperCode(node);
      case 'lower':
        return this.generateLowerCode(node);
      case 'date':
        return this.generateDateCode(node);
      case 'now':
        return this.generateNowCode(node);
      case 'today':
        return this.generateTodayCode(node);
      case 'dateAdd':
        return this.generateDateAddCode(node);
      case 'formatDate':
        return this.generateFormatDateCode(node);
      case 'encrypt':
        return this.generateEncryptCode(node);
      case 'decrypt':
        return this.generateDecryptCode(node);
      case 'hash256':
        return this.generateHash256Code(node);
      case 'sign':
        return this.generateSignCode(node);
      case 'LogPrint':
      case 'Log Print':
      case 'logPrint':
        return this.generateLogPrintCode(node);
      case 'Create Transform':
        return this.generateCreateTransformCode(node);
      case 'Add Mapping':
        return this.generateAddMappingCode(node);
      case 'Add Mapping Transform':
      case 'AddMappingWithTransform':
      case 'addMappingWithTransform':
        return this.generateAddMappingWithTransformCode(node);
      case 'Do ETL':
      case 'doETL':
        return this.generateDoETLCode(node);
      case 'ETL Status':
      case 'etlStatus':
        return this.generateETLStatusCode(node);
      case 'Get Transform':
      case 'GetTransform':
      case 'getTransform':
        return this.generateGetTransformCode(node);
      case 'List Transforms':
      case 'ListTransforms':
      case 'listTransforms':
        return this.generateListTransformsCode(node);
      case 'Register Transform':
      case 'RegisterTransform':
      case 'registerTransform':
        return this.generateRegisterTransformCode(node);
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
      case 'and':
        return this.generateAndCode(node);
      case 'or':
        return this.generateOrCode(node);
      case 'not':
        return this.generateNotCode(node);
      case 'equal':
        return this.generateEqualCode(node);
      case 'unequal':
        return this.generateUnequalCode(node);
      case 'bigger':
        return this.generateBiggerCode(node);
      case 'biggerEq':
        return this.generateBiggerEqCode(node);
      case 'smaller':
        return this.generateSmallerCode(node);
      case 'smallerEq':
        return this.generateSmallerEqCode(node);
      case 'add':
        return this.generateAddCode(node);
      case 'sub':
        return this.generateSubCode(node);
      case 'mul':
        return this.generateMulCode(node);
      case 'div':
        return this.generateDivCode(node);
      case 'abs':
        return this.generateAbsCode(node);
      case 'max':
        return this.generateMaxCode(node);
      case 'min':
        return this.generateMinCode(node);
      case 'round':
        return this.generateRoundCode(node);
      case 'random':
        return this.generateRandomCode(node);
      case 'Exists':
      case 'exists':
        return this.generateExistsCode(node);
      case 'Type Of':
      case 'typeOf':
        return this.generateTypeOfCode(node);
      case 'Value Of':
      case 'valueOf':
        return this.generateValueOfCode(node);
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
      case 'Sleep':
        return this.generateSleepCode(node);
      case 'Get Env':
        return this.generateGetEnvCode(node);
      case 'Exit':
        return this.generateExitCode(node);
      case 'Call Method':
        return this.generateCallMethodCode(node);
      case 'Get Host Object':
        return this.generateGetHostObjectCode(node);
      case 'Host Object':
        return this.generateHostObjectCode(node);
      case 'Apply':
        return this.generateApplyCode(node);
      case 'Clone':
        return this.generateCloneCode(node);
      case 'Contains':
        return this.generateContainsCode(node);
      case 'Get All Meta':
        return this.generateGetAllMetaCode(node);
      case 'Get At':
        return this.generateGetAtCode(node);
      case 'Get Attributes':
        return this.generateGetAttributesCode(node);
      case 'Get Meta':
        return this.generateGetMetaCode(node);
      case 'Get Property':
        return this.generateGetPropCode(node);
      case 'Set Meta':
        return this.generateSetMetaCode(node);
      case 'Set Property':
        return this.generateSetPropCode(node);
      case 'Index Of':
        return this.generateIndexOfCode(node);
      default:
        return 'Error - unknown function'; // this.generateGenericFunctionCode(node);
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
        const inlineLabels = ['Create', 'New Tree', 'Parse JSON', 'parseJSON', 'parseJSONSimple', 'Array', 'Range'];
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

  private generateParseJSONSimpleCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let jsonString = props.jsonString || '{}';
    if (jsonString === '{ [] }') {
      jsonString = '[]';
    } else if (jsonString === '{ ["admin", "contributor", "viewer"] }') {
      jsonString = '["admin", "contributor", "viewer"]';
    }
    return `parseJSONSimple('${jsonString}')`;
  }

  private generateToJSONCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = props.value || 'myValue';
    return `toJSON(${value})`;
  }

  private generateToSimpleJSONCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = props.value || 'myValue';
    return `toSimpleJSON(${value})`;
  }

  private generateCSVHeadersCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const nodeOrPath = props.nodeOrPath || 'csvNode';
    // If it looks like a path (contains / or . or starts with quote), quote it
    const param = nodeOrPath.includes('/') || nodeOrPath.includes('.') || nodeOrPath.startsWith("'") ? 
      (nodeOrPath.startsWith("'") ? nodeOrPath : `'${nodeOrPath}'`) : nodeOrPath;
    return `csvHeaders(${param})`;
  }

  private generateCSVRowCountCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const nodeOrPath = props.nodeOrPath || 'csvNode';
    const param = nodeOrPath.includes('/') || nodeOrPath.includes('.') || nodeOrPath.startsWith("'") ? 
      (nodeOrPath.startsWith("'") ? nodeOrPath : `'${nodeOrPath}'`) : nodeOrPath;
    return `csvRowCount(${param})`;
  }

  private generateCSVColumnCountCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const nodeOrPath = props.nodeOrPath || 'csvNode';
    const param = nodeOrPath.includes('/') || nodeOrPath.includes('.') || nodeOrPath.startsWith("'") ? 
      (nodeOrPath.startsWith("'") ? nodeOrPath : `'${nodeOrPath}'`) : nodeOrPath;
    return `csvColumnCount(${param})`;
  }

  private generateCSVGetRowCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const nodeOrPath = props.nodeOrPath || 'csvNode';
    const index = props.index || '0';
    const param = nodeOrPath.includes('/') || nodeOrPath.includes('.') || nodeOrPath.startsWith("'") ? 
      (nodeOrPath.startsWith("'") ? nodeOrPath : `'${nodeOrPath}'`) : nodeOrPath;
    return `csvGetRow(${param}, ${index})`;
  }

  private generateCSVGetCellCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const nodeOrPath = props.nodeOrPath || 'csvNode';
    const rowIndex = props.rowIndex || '0';
    const colIndexOrName = props.colIndexOrName || '0';
    const param = nodeOrPath.includes('/') || nodeOrPath.includes('.') || nodeOrPath.startsWith("'") ? 
      (nodeOrPath.startsWith("'") ? nodeOrPath : `'${nodeOrPath}'`) : nodeOrPath;
    // If colIndexOrName is a string (column name), quote it
    const colParam = isNaN(Number(colIndexOrName)) && !colIndexOrName.startsWith("'") ? `'${colIndexOrName}'` : colIndexOrName;
    return `csvGetCell(${param}, ${rowIndex}, ${colParam})`;
  }

  private generateCSVGetRowsCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const nodeOrPath = props.nodeOrPath || 'csvNode';
    const param = nodeOrPath.includes('/') || nodeOrPath.includes('.') || nodeOrPath.startsWith("'") ? 
      (nodeOrPath.startsWith("'") ? nodeOrPath : `'${nodeOrPath}'`) : nodeOrPath;
    return `csvGetRows(${param})`;
  }

  private generateCSVToCSVCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const nodeOrPath = props.nodeOrPath || 'csvNode';
    const param = nodeOrPath.includes('/') || nodeOrPath.includes('.') || nodeOrPath.startsWith("'") ? 
      (nodeOrPath.startsWith("'") ? nodeOrPath : `'${nodeOrPath}'`) : nodeOrPath;
    return `csvToCSV(${param})`;
  }

  private generateCSVLoadCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const csvNode = props.node || 'csvNode';
    const path = props.path || 'data/file.csv';
    // Path should always be quoted
    const pathParam = path.startsWith("'") ? path : `'${path}'`;
    return `csvLoad(${csvNode}, ${pathParam})`;
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

  private generateSleepCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const milliseconds = props.milliseconds || '1000';
    // Sleep takes a numeric argument - don't quote it
    return `sleep(${milliseconds})`;
  }

  private generateGetEnvCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const varName = props.varName || 'PATH';
    // Environment variable name should be quoted as a string literal
    return `getEnv('${varName}')`;
  }

  private generateExitCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const exitCode = props.exitCode;
    // Exit takes optional numeric code - don't quote it, or omit if 0/empty
    if (!exitCode || exitCode === '0') {
      return 'exit()';
    }
    return `exit(${exitCode})`;
  }

  private generateCallMethodCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const objectName = props.objectName || 'obj';
    const methodName = props.methodName || 'method';
    const args = props.args || '';
    
    // Format: callMethod(objOrName, 'methodName', args...)
    const argsList = args ? `, ${args}` : '';
    
    // If objectName looks like a variable, don't quote it; otherwise quote it
    const objRef = /^[a-zA-Z_][a-zA-Z0-9_]*$/.test(objectName) 
      ? objectName 
      : `'${objectName}'`;
    
    return `callMethod(${objRef}, '${methodName}'${argsList})`;
  }

  private generateGetHostObjectCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const objectName = props.objectName || 'obj';
    // Format: getHostObject('name')
    return `getHostObject('${objectName}')`;
  }

  private generateHostObjectCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const objectName = props.objectName || 'obj';
    const wrappedObject = props.wrappedObject || '';
    
    // Format: hostObject('name') or hostObject('name', obj)
    if (wrappedObject) {
      return `hostObject('${objectName}', ${wrappedObject})`;
    }
    return `hostObject('${objectName}')`;
  }

  private generateApplyCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const functionName = props.functionName || 'func';
    const collection = props.collection || 'collection';
    // Format: apply(fn, collection)
    return `apply(${functionName}, ${collection})`;
  }

  private generateCloneCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const object = props.object || 'obj';
    const newName = props.newName || '';
    // Format: clone(obj) or clone(obj, 'newName')
    if (newName) {
      return `clone(${object}, '${newName}')`;
    }
    return `clone(${object})`;
  }

  private generateContainsCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const object = props.object || 'str';
    const value = props.value || 'value';
    // Format: contains(obj, value)
    // If value looks like a string literal, keep quotes; if variable, no quotes
    const valueArg = value.startsWith("'") || value.startsWith('"') 
      ? value 
      : `'${value}'`;
    return `contains(${object}, ${valueArg})`;
  }

  private generateGetAllMetaCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const target = props.target || 'obj';
    return `getAllMeta(${target})`;
  }

  private generateGetAtCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const target = props.target || 'obj';
    const index = props.index || '0';
    return `getAt(${target}, ${index})`;
  }

  private generateGetAttributesCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const target = props.target || 'obj';
    return `getAttributes(${target})`;
  }

  private generateGetMetaCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const target = props.target || 'obj';
    const key = props.key || 'metaKey';
    const keyArg = key.startsWith("'") || key.startsWith('"') ? key : `'${key}'`;
    return `getMeta(${target}, ${keyArg})`;
  }

  private generateGetPropCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const target = props.target || 'obj';
    const key = props.key || 'key';
    const keyArg = key.startsWith("'") || key.startsWith('"') ? key : `'${key}'`;
    return `getProp(${target}, ${keyArg})`;
  }

  private generateIndexOfCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const target = props.target || 'obj';
    const value = props.value || 'value';
    const startIndex = props.startIndex || '';
    const valueArg = value.startsWith("'") || value.startsWith('"') ? value : `'${value}'`;
    if (startIndex && startIndex.trim().length > 0) {
      return `indexOf(${target}, ${valueArg}, ${startIndex})`;
    }
    return `indexOf(${target}, ${valueArg})`;
  }

  private generateSetMetaCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const targetRaw = (props.target ?? 'obj').toString().trim();
    const target = targetRaw !== '' ? targetRaw : 'obj';
    const keyRaw = (props.key ?? 'metaKey').toString().trim();
    const key = keyRaw !== '' ? keyRaw : 'metaKey';
    const keyArg = key.startsWith("'") || key.startsWith('"') ? key : `'${key}'`;
    const valueArg = this.formatSetterValue(props.value);
    return `setMeta(${target}, ${keyArg}, ${valueArg})`;
  }

  private generateSetPropCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const targetRaw = (props.target ?? 'obj').toString().trim();
    const target = targetRaw !== '' ? targetRaw : 'obj';
    const keyRaw = (props.key ?? 'key').toString().trim();
    const key = keyRaw !== '' ? keyRaw : 'key';
    const keyArg = key.startsWith("'") || key.startsWith('"') ? key : `'${key}'`;
    const valueArg = this.formatSetterValue(props.value);
    return `setProp(${target}, ${keyArg}, ${valueArg})`;
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

  private generateAddMappingWithTransformCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    let transform = (props.transform ?? 'transform').toString().trim() || 'transform';
    if ((transform.startsWith('"') && transform.endsWith('"')) || (transform.startsWith("'") && transform.endsWith("'"))) {
      transform = transform.slice(1, -1);
    }

    const literal = (value: unknown, fallback: string) => {
      const raw = (value ?? fallback).toString().trim() || fallback;
      if ((raw.startsWith("'") && raw.endsWith("'")) || (raw.startsWith('"') && raw.endsWith('"'))) {
        return raw;
      }
      return `'${raw.replace(/'/g, "\\'")}'`;
    };

    const sourceField = literal(props.sourceField, 'sourceField');
    const targetColumn = literal(props.targetColumn, 'targetColumn');
    const transformName = literal(props.transformName, 'transformName');
    const dataType = literal(props.dataType, 'string');
    const required = props.required !== undefined ? props.required : false;
    const defaultValueRaw = (props.defaultValue ?? '').toString().trim();

    const args = [transform, sourceField, targetColumn, transformName, dataType, String(required)];
    if (defaultValueRaw.length > 0) {
      args.push(literal(defaultValueRaw, defaultValueRaw));
    }
    return `addMappingWithTransform(${args.join(', ')})`;
  }

  private generateDoETLCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const jobIdArg = this.formatSetterValue(props.jobId ?? "'etl_job'");
    const csvFileArg = this.formatSetterValue(props.csvFile ?? "'data.csv'");
    const transformConfig = (props.transformConfig ?? 'transformConfig').toString().trim() || 'transformConfig';
    const targetConfig = (props.targetConfig ?? 'targetConfig').toString().trim() || 'targetConfig';
    const optionsRaw = (props.options ?? '').toString().trim();

    const args = [jobIdArg, csvFileArg, transformConfig, targetConfig];
    if (optionsRaw.length > 0) {
      args.push(optionsRaw);
    }
    return `doETL(${args.join(', ')})`;
  }

  private generateETLStatusCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const jobIdArg = this.formatSetterValue(props.jobId ?? "'etl_job'");
    return `etlStatus(${jobIdArg})`;
  }

  private generateGetTransformCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const nameArg = this.formatSetterValue(props.transformName ?? "'transformName'");
    return `getTransform(${nameArg})`;
  }

  private generateListTransformsCode(_node: VisualDSLNode): string {
    return 'listTransforms()';
  }

  private generateRegisterTransformCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const literal = (value: unknown, fallback: string) => {
      const rawInput = (value ?? fallback).toString().trim();
      const resolved = rawInput.length > 0 ? rawInput : fallback;
      if (resolved.length === 0) {
        return "''";
      }
      if ((resolved.startsWith("'") && resolved.endsWith("'")) || (resolved.startsWith('"') && resolved.endsWith('"'))) {
        return resolved;
      }
      return `'${resolved.replace(/'/g, "\\'")}'`;
    };

    const nameArg = this.formatSetterValue(props.transformName ?? "'transformName'");
    const entries: string[] = [];

    const descriptionRaw = (props.description ?? '').toString().trim();
    if (descriptionRaw.length > 0) {
      entries.push(`'description', ${literal(descriptionRaw, descriptionRaw)}`);
    }

    const dataTypeRaw = (props.dataType ?? 'string').toString().trim();
    const dataTypeValue = dataTypeRaw.length > 0 ? dataTypeRaw : 'string';
    entries.push(`'dataType', ${literal(dataTypeValue, dataTypeValue)}`);

    const categoryRaw = (props.category ?? '').toString().trim();
    if (categoryRaw.length > 0) {
      entries.push(`'category', ${literal(categoryRaw, categoryRaw)}`);
    }

    const programLines = Array.isArray(props.program) ? props.program : [];
    const sanitizedProgram = programLines
      .map(line => (line ?? '').toString().trim())
      .filter(line => line.length > 0)
      .map(line => `'${line.replace(/'/g, "\\'")}'`);
    const programExpr = sanitizedProgram.length > 0 ? `array(${sanitizedProgram.join(', ')})` : 'array()';
    entries.push(`'program', ${programExpr}`);

    const config = entries.length > 0 ? `map(${entries.join(', ')})` : 'map()';
    return `registerTransform(${nameArg}, ${config})`;
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

  private generateAndCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const operands = this.normalizeLogicOperands(props.operands, 2, 'true');
    return `and(${operands.join(', ')})`;
  }

  private generateOrCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const operands = this.normalizeLogicOperands(props.operands, 2, 'false');
    return `or(${operands.join(', ')})`;
  }

  private generateNotCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const operands = this.normalizeLogicOperands(props.operands ?? props.operand, 1, 'flag');
    return `not(${operands.join(', ')})`;
  }

  private generateEqualCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const legacyLeft = this.coerceExpression(props.leftOperand, 'valueA');
    const legacyRight = this.coerceExpression(props.rightOperand, 'valueB');
    const rawOperands = Array.isArray(props.operands) ? props.operands : [legacyLeft, legacyRight];
    const operands = this.normalizeLogicOperands(rawOperands, 2, legacyRight);
    return `equal(${operands.join(', ')})`;
  }

  private generateUnequalCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const legacyLeft = this.coerceExpression(props.leftOperand, 'valueA');
    const legacyRight = this.coerceExpression(props.rightOperand, 'valueB');
    const rawOperands = Array.isArray(props.operands) ? props.operands : [legacyLeft, legacyRight];
    const operands = this.normalizeLogicOperands(rawOperands, 2, legacyRight);
    return `unequal(${operands.join(', ')})`;
  }

  private generateBiggerCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const left = this.coerceExpression(props.leftOperand, 'valueA');
    const right = this.coerceExpression(props.rightOperand, 'valueB');
    return `bigger(${left}, ${right})`;
  }

  private generateBiggerEqCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const left = this.coerceExpression(props.leftOperand, 'valueA');
    const right = this.coerceExpression(props.rightOperand, 'valueB');
    return `biggerEq(${left}, ${right})`;
  }

  private generateSmallerCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const left = this.coerceExpression(props.leftOperand, 'valueA');
    const right = this.coerceExpression(props.rightOperand, 'valueB');
    return `smaller(${left}, ${right})`;
  }

  private generateSmallerEqCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const left = this.coerceExpression(props.leftOperand, 'valueA');
    const right = this.coerceExpression(props.rightOperand, 'valueB');
    return `smallerEq(${left}, ${right})`;
  }

  private generateAddCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const [left, right] = this.resolveBinaryMathOperands(props, 'valueA', 'valueB');
    return `add(${left}, ${right})`;
  }

  private generateSubCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const [left, right] = this.resolveBinaryMathOperands(props, 'valueA', 'valueB');
    return `sub(${left}, ${right})`;
  }

  private generateMulCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const [left, right] = this.resolveBinaryMathOperands(props, 'valueA', 'valueB');
    return `mul(${left}, ${right})`;
  }

  private generateDivCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const [left, right] = this.resolveBinaryMathOperands(props, 'numerator', 'denominator');
    return `div(${left}, ${right})`;
  }

  private generateAbsCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const operands = this.normalizeExpressionList((props as { operands?: unknown }).operands);
    const source = operands.length > 0 ? operands[0] : (props as { operand?: unknown; value?: unknown }).operand ?? props.value;
    const value = this.coerceExpression(source, 'value');
    return `abs(${value})`;
  }

  private generateMaxCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const values = this.normalizeMathList((props as { operands?: unknown }).operands, ['valueA', 'valueB']);
    return `max(${values.join(', ')})`;
  }

  private generateMinCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const values = this.normalizeMathList((props as { operands?: unknown }).operands, ['valueA', 'valueB']);
    return `min(${values.join(', ')})`;
  }

  private generateRoundCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const operands = this.normalizeExpressionList((props as { operands?: unknown }).operands);
    const valueSource = operands.length > 0 ? operands[0] : (props as { value?: unknown; operand?: unknown }).value ?? props.operand;
    const value = this.coerceExpression(valueSource, 'value');
    const decimalsRaw = (props as { decimalPlaces?: unknown }).decimalPlaces ?? '';
    const decimalsText = decimalsRaw.toString().trim();
    if (decimalsText.length === 0) {
      return `round(${value})`;
    }
    const decimals = this.coerceExpression(decimalsRaw, '0');
    return `round(${value}, ${decimals})`;
  }

  private generateRandomCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const operands = this.normalizeExpressionList((props as { operands?: unknown }).operands);
    if (operands.length === 0) {
      return 'random()';
    }
    if (operands.length === 1) {
      return `random(${operands[0]})`;
    }
    return `random(${operands[0]}, ${operands[1]})`;
  }

  private generateConcatCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const normalized = this.normalizeExpressionList((props as { operands?: unknown }).operands);
    const operands = normalized.length > 0 ? normalized : ['valueA', 'valueB'];
    return `concat(${operands.join(', ')})`;
  }

  private generateSplitCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = this.coerceStringArgument((props as { value?: unknown }).value, 'textValue');
    const delimiter = this.coerceStringArgument((props as { delimiter?: unknown }).delimiter, "','");
    return `split(${value}, ${delimiter})`;
  }

  private generateReplaceCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = this.coerceStringArgument((props as { value?: unknown }).value, 'textValue');
    const searchValue = this.coerceStringArgument((props as { searchValue?: unknown }).searchValue, 'oldText');
    const replaceValue = this.coerceStringArgument((props as { replaceValue?: unknown }).replaceValue, 'newText');
    const countRaw = (props as { count?: unknown }).count ?? '';
    const countText = countRaw.toString().trim();
    if (countText.length === 0) {
      return `replace(${value}, ${searchValue}, ${replaceValue})`;
    }
    const count = this.coerceExpression(countRaw, '-1');
    return `replace(${value}, ${searchValue}, ${replaceValue}, ${count})`;
  }

  private generateSubstringCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = this.coerceStringArgument((props as { value?: unknown }).value, 'textValue');
    const start = this.coerceExpression((props as { start?: unknown }).start, '0');
    const lengthRaw = (props as { length?: unknown }).length ?? '';
    const lengthText = lengthRaw.toString().trim();
    if (lengthText.length === 0) {
      return `substring(${value}, ${start})`;
    }
    const lengthExpr = this.coerceExpression(lengthRaw, '0');
    return `substring(${value}, ${start}, ${lengthExpr})`;
  }

  private generateStringLengthCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = this.coerceStringArgument((props as { value?: unknown }).value, 'textValue');
    return `strlen(${value})`;
  }

  private generateUpperCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = this.coerceStringArgument((props as { value?: unknown }).value, 'textValue');
    return `upper(${value})`;
  }

  private generateLowerCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = this.coerceStringArgument((props as { value?: unknown }).value, 'textValue');
    return `lower(${value})`;
  }

  private generateDateCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const modeRaw = (props as { mode?: unknown }).mode;
    const normalizedMode = typeof modeRaw === 'string' ? modeRaw.toLowerCase() : '';
    const hasComponentFallback = ['year', 'month', 'day'].some((key) => {
      const value = (props as Record<string, unknown>)[key];
      return value !== undefined && value !== null && value.toString().trim().length > 0;
    });
    if (normalizedMode === 'components' || (normalizedMode === '' && hasComponentFallback)) {
      const year = this.coerceExpression((props as { year?: unknown }).year, '2024');
      const month = this.coerceExpression((props as { month?: unknown }).month, '1');
      const day = this.coerceExpression((props as { day?: unknown }).day, '1');
      return `date(${year}, ${month}, ${day})`;
    }
    const value = this.coerceStringArgument((props as { value?: unknown }).value, '2024-01-01T00:00:00Z');
    return `date(${value})`;
  }

  private generateNowCode(_node: VisualDSLNode): string {
    return 'now()';
  }

  private generateTodayCode(_node: VisualDSLNode): string {
    return 'today()';
  }

  private generateDateAddCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = this.coerceStringArgument((props as { value?: unknown }).value, 'now()');
    const interval = this.coerceIntervalArgument((props as { interval?: unknown }).interval, 'day');
    const amount = this.coerceExpression((props as { amount?: unknown }).amount, '1');
    return `dateAdd(${value}, ${interval}, ${amount})`;
  }

  private generateFormatDateCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = this.coerceStringArgument((props as { value?: unknown }).value, 'now()');
    const formatRaw = (props as { format?: unknown }).format ?? '';
    const formatText = formatRaw.toString().trim();
    if (formatText.length === 0) {
      return `formatDate(${value})`;
    }
    const format = this.coerceStringArgument(formatRaw, 'YYYY-MM-DD');
    return `formatDate(${value}, ${format})`;
  }

  private generateEncryptCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const keyId = this.coerceStringArgument((props as { keyId?: unknown }).keyId, 'encKey');
    const data = this.coerceStringArgument((props as { data?: unknown }).data, 'plaintextValue');
    return `encrypt(${keyId}, ${data})`;
  }

  private generateDecryptCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const keyId = this.coerceStringArgument((props as { keyId?: unknown }).keyId, 'encKey');
    const ciphertext = this.coerceStringArgument((props as { ciphertext?: unknown }).ciphertext, 'ciphertextBase64');
    return `decrypt(${keyId}, ${ciphertext})`;
  }

  private generateHash256Code(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = this.coerceStringArgument((props as { value?: unknown }).value, 'textValue');
    return `hash256(${value})`;
  }

  private generateSignCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const keyId = this.coerceStringArgument((props as { keyId?: unknown }).keyId, 'encKey');
    const data = this.coerceStringArgument((props as { data?: unknown }).data, 'message');
    return `sign(${keyId}, ${data})`;
  }

  private generateExistsCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const variableName = props.variableName || 'myVar';
    return `exists('${variableName}')`;
  }

  private generateTypeOfCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = props.value || 'myVar';
    return `typeOf(${value})`;
  }

  private generateValueOfCode(node: VisualDSLNode): string {
    const props = node.data.properties || {};
    const value = props.value || 'myVar';
    const targetType = props.targetType;
    if (targetType) {
      return `valueOf(${value}, '${targetType}')`;
    }
    return `valueOf(${value})`;
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
      'addMappingWithTransform': 'addMappingWithTransform',
      'addmappingwithtransform': 'addMappingWithTransform',
      'add mapping transform': 'addMappingWithTransform',
      'add mapping with transform': 'addMappingWithTransform',
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
      'doETL': 'doETL',
      'doetl': 'doETL',
      'do etl': 'doETL',
      'etlStatus': 'etlStatus',
      'etlstatus': 'etlStatus',
      'etl status': 'etlStatus',
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
      case 'Sleep':
        return null; // sleep() doesn't return a value
      case 'Exit':
        return null; // exit() doesn't return a value
      case 'Get Env':
        return this.generateGetEnvCode(childNode); // getEnv() can be inlined
      case 'Call Method':
        return this.generateCallMethodCode(childNode); // callMethod() can be inlined
      case 'Get Host Object':
        return this.generateGetHostObjectCode(childNode); // getHostObject() can be inlined
      case 'Host Object':
        return this.generateHostObjectCode(childNode); // hostObject() can be inlined
      case 'Apply':
        return null; // apply() doesn't return a usable value for inline
      case 'Clone':
        return this.generateCloneCode(childNode); // clone() can be inlined
      case 'Contains':
        return this.generateContainsCode(childNode); // contains() returns boolean, can be inlined
      case 'Get All Meta':
        return this.generateGetAllMetaCode(childNode);
      case 'Get At':
        return this.generateGetAtCode(childNode);
      case 'Get Attributes':
        return this.generateGetAttributesCode(childNode);
      case 'Get Meta':
        return this.generateGetMetaCode(childNode);
      case 'Get Property':
        return this.generateGetPropCode(childNode);
      case 'Index Of':
        return this.generateIndexOfCode(childNode);
      case 'and':
        return this.generateAndCode(childNode);
      case 'or':
        return this.generateOrCode(childNode);
      case 'not':
        return this.generateNotCode(childNode);
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
    const valueArg = this.formatSetterValue(props.value);
    return `setAttribute(${varName}, '${attributeName}', ${valueArg})`;
  }

  private formatSetterValue(value: unknown): string {
    const rawOriginal = value ?? '';
    const raw = rawOriginal.toString().trim();
    if (raw === '') {
      return `''`;
    }
    if ((raw.startsWith("'") && raw.endsWith("'")) || (raw.startsWith('"') && raw.endsWith('"'))) {
      return raw;
    }
    if (/^func\s*\(/.test(raw)) {
      return raw;
    }
    if (raw === 'true' || raw === 'false') {
      return raw;
    }
    if (!isNaN(Number(raw)) && /^-?\d+(\.\d+)?$/.test(raw)) {
      return raw;
    }
    if (/^[A-Za-z_][A-Za-z0-9_]*$/.test(raw) || /\)$/.test(raw)) {
      return raw;
    }
    const escaped = raw.replace(/'/g, "\\'");
    return `'${escaped}'`;
  }

  private normalizeExpressionList(raw: unknown): string[] {
    if (Array.isArray(raw)) {
      return raw
        .map(entry => (entry ?? '').toString().trim())
        .filter(entry => entry.length > 0);
    }
    if (raw === undefined || raw === null) {
      return [];
    }
    const text = raw.toString().trim();
    return text.length > 0 ? [text] : [];
  }

  private normalizeLogicOperands(raw: unknown, minLength: number, padValue: string): string[] {
    const min = Math.max(minLength, 1);
    const operands = this.normalizeExpressionList(raw);
    if (operands.length === 0) {
      return Array(min).fill(padValue);
    }
    const normalized = [...operands];
    while (normalized.length < min) {
      normalized.push(normalized[normalized.length - 1]);
    }
    return normalized;
  }

  private coerceExpression(value: unknown, fallback: string): string {
    const text = (value ?? '').toString().trim();
    return text.length > 0 ? text : fallback;
  }

  private coerceStringArgument(value: unknown, fallback: string): string {
    const raw = (value ?? '').toString().trim();
    const candidate = raw.length > 0 ? raw : fallback;
    const hasWrappingQuotes =
      (candidate.startsWith('"') && candidate.endsWith('"') && candidate.length >= 2) ||
      (candidate.startsWith("'") && candidate.endsWith("'") && candidate.length >= 2);
    if (hasWrappingQuotes) {
      const inner = candidate.slice(1, -1).replace(/'/g, "\\'");
      return `'${inner}'`;
    }
    if (this.isLikelyExpression(candidate)) {
      return candidate;
    }
    const escaped = candidate.replace(/'/g, "\\'");
    return `'${escaped}'`;
  }

  private coerceIntervalArgument(value: unknown, fallback: string): string {
    const raw = (value ?? '').toString().trim();
    const candidate = raw.length > 0 ? raw : fallback;
    const hasWrappingQuotes =
      (candidate.startsWith('"') && candidate.endsWith('"') && candidate.length >= 2) ||
      (candidate.startsWith("'") && candidate.endsWith("'") && candidate.length >= 2);
    const normalized = candidate.toLowerCase();
    const knownIntervals = ['year', 'years', 'month', 'months', 'day', 'days', 'hour', 'hours', 'minute', 'minutes', 'second', 'seconds'];
    if (hasWrappingQuotes) {
      const inner = candidate.slice(1, -1).replace(/'/g, "\\'");
      return `'${inner}'`;
    }
    if (knownIntervals.includes(normalized)) {
      const escaped = candidate.replace(/'/g, "\\'");
      return `'${escaped}'`;
    }
    if (this.isLikelyExpression(candidate)) {
      return candidate;
    }
    const escaped = candidate.replace(/'/g, "\\'");
    return `'${escaped}'`;
  }

  private isLikelyExpression(value: string): boolean {
    if (!value) {
      return false;
    }
    if (/^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$/.test(value)) {
      return true;
    }
    if (/^-?\d+(\.\d+)?$/.test(value)) {
      return true;
    }
    if (value.includes('(') || value.includes('[') || value.includes('{')) {
      return true;
    }
    if (value === 'DBNull') {
      return true;
    }
    return false;
  }

  private resolveBinaryMathOperands(
    props: Record<string, unknown>,
    defaultLeft: string,
    defaultRight: string
  ): [string, string] {
    const candidate = props as {
      operands?: unknown;
      leftOperand?: unknown;
      rightOperand?: unknown;
    };
    const arrayOperands = Array.isArray(candidate.operands) ? candidate.operands : null;
    const leftSource = arrayOperands && arrayOperands.length > 0 ? arrayOperands[0] : candidate.leftOperand;
    const rightSource = arrayOperands && arrayOperands.length > 1 ? arrayOperands[1] : candidate.rightOperand;
    const left = this.coerceExpression(leftSource, defaultLeft);
    const right = this.coerceExpression(rightSource, defaultRight);
    return [left, right];
  }

  private normalizeMathList(raw: unknown, fallbackValues: string[]): string[] {
    const normalized = this.normalizeExpressionList(raw);
    if (normalized.length === 0) {
      return [...fallbackValues];
    }
    return normalized;
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
