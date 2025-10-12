import React from 'react';
import { ReactFlow, ReactFlowProvider, Node, Edge, Connection, addEdge, applyNodeChanges, applyEdgeChanges, NodeChange, EdgeChange, reconnectEdge, MiniMap, Controls, Background, BackgroundVariant } from 'reactflow';
import { flushSync } from 'react-dom';
import "reactflow/dist/style.css";

import { LogiconPalette } from "../components/LogiconPalette";
import { LogiconNode } from "../components/LogiconNode";
import { GroupNode } from "../components/GroupNode";
import SubflowFrame from "../components/SubflowFrame";
import { ContextMenu } from "../components/ContextMenu";
import { DiagramToolbar } from "../components/DiagramToolbar";
import { StartNodePropertiesDialog, StartNodeProperties } from "../components/dialogs/StartNodeProperties";
import { DeclareNodePropertiesDialog, DeclareNodeProperties } from "../components/dialogs/DeclareNodeProperties";
import { CreateNodePropertiesDialog, CreateNodeProperties } from "../components/dialogs/CreateNodeProperties";
import { NewTreeNodePropertiesDialog, NewTreeNodeProperties } from "../components/dialogs/NewTreeNodeProperties";
import { CSVNodePropertiesDialog, CSVNodeProperties } from "../components/dialogs/CSVNodeProperties";
import { JSONNodePropertiesDialog, JSONNodeProperties } from "../components/dialogs/JSONNodeProperties";
import { MapNodePropertiesDialog, MapNodeProperties } from "../components/dialogs/MapNodeProperties";
import { NodeToStringNodePropertiesDialog, NodeToStringNodeProperties } from "../components/dialogs/NodeToStringNodeProperties";
import { QueryNodePropertiesDialog, QueryNodeProperties } from "../components/dialogs/QueryNodeProperties";
import { ListNodePropertiesDialog, ListNodeProperties } from "../components/dialogs/ListNodeProperties";
import { FindByNameNodePropertiesDialog, FindByNameNodeProperties } from "../components/dialogs/FindByNameNodeProperties";
import { FirstChildNodePropertiesDialog, FirstChildNodeProperties } from "../components/dialogs/FirstChildNodeProperties";
import { LastChildNodePropertiesDialog, LastChildNodeProperties } from "../components/dialogs/LastChildNodeProperties";
import { GetAttributeNodePropertiesDialog, GetAttributeNodeProperties } from "../components/dialogs/GetAttributeNodeProperties";
import { RemoveAttributeNodePropertiesDialog, RemoveAttributeNodeProperties } from "../components/dialogs/RemoveAttributeNodeProperties";
import SetAttributeNodePropertiesDialog, { SetAttributeNodeProperties as SetAttributeProps } from "../components/dialogs/SetAttributeNodeProperties";
import { SetAttributesNodePropertiesDialog, SetAttributesNodeProperties } from "../components/dialogs/SetAttributesNodeProperties";
import { GetChildAtNodePropertiesDialog, GetChildAtNodeProperties } from "../components/dialogs/GetChildAtNodeProperties";
import { GetChildByNameNodePropertiesDialog, GetChildByNameNodeProperties } from "../components/dialogs/GetChildByNameNodeProperties";
import { GetDepthNodePropertiesDialog, GetDepthNodeProperties } from "../components/dialogs/GetDepthNodeProperties";
import { GetLevelNodePropertiesDialog, GetLevelNodeProperties } from "../components/dialogs/GetLevelNodeProperties";
import { GetNameNodePropertiesDialog, GetNameNodeProperties } from "../components/dialogs/GetNameNodeProperties";
import { SetNameNodePropertiesDialog, SetNameNodeProperties } from "../components/dialogs/SetNameNodeProperties";
import { GetParentNodePropertiesDialog, GetParentNodeProperties } from "../components/dialogs/GetParentNodeProperties";
import { GetPathNodePropertiesDialog, GetPathNodeProperties } from "../components/dialogs/GetPathNodeProperties";
import { GetRootNodePropertiesDialog, GetRootNodeProperties } from "../components/dialogs/GetRootNodeProperties";
import { GetSiblingsNodePropertiesDialog, GetSiblingsNodeProperties } from "../components/dialogs/GetSiblingsNodeProperties";
import { GetTextNodePropertiesDialog, GetTextNodeProperties } from "../components/dialogs/GetTextNodeProperties";
import { SetTextNodePropertiesDialog, SetTextNodeProperties } from "../components/dialogs/SetTextNodeProperties";
import { HasAttributeNodePropertiesDialog, HasAttributeNodeProperties } from "../components/dialogs/HasAttributeNodeProperties";
import { IsLeafNodePropertiesDialog, IsLeafNodeProperties } from "../components/dialogs/IsLeafNodeProperties";
import { IsRootNodePropertiesDialog, IsRootNodeProperties } from "../components/dialogs/IsRootNodeProperties";
import { ParseJSONNodePropertiesDialog, ParseJSONNodeProperties } from "../components/dialogs/ParseJSONNodeProperties";
import { ArrayNodePropertiesDialog, ArrayNodeProperties } from "../components/dialogs/ArrayNodeProperties";
import { AddChildNodePropertiesDialog, AddChildNodeProperties } from "../components/dialogs/AddChildNodeProperties";
import { RemoveChildNodePropertiesDialog, RemoveChildNodeProperties } from "../components/dialogs/RemoveChildNodeProperties";
import { ChildCountNodePropertiesDialog, ChildCountNodeProperties } from "../components/dialogs/ChildCountNodeProperties";
import { ClearNodePropertiesDialog, ClearNodeProperties } from "../components/dialogs/ClearNodeProperties";
import { AddToNodePropertiesDialog, AddToNodeProperties } from "../components/dialogs/AddToNodeProperties";
import LogPrintNodeProperties, { LogPrintNodeProperties as LogPrintProperties } from "../components/dialogs/LogPrintNodeProperties";
import CreateTransformNodeProperties, { CreateTransformNodeProperties as CreateTransformProperties } from "../components/dialogs/CreateTransformNodeProperties";
import AddMappingNodeProperties, { AddMappingNodeProperties as AddMappingProperties } from "../components/dialogs/AddMappingNodeProperties";
import { TreeSaveNodePropertiesDialog, TreeSaveNodeProperties } from "../components/dialogs/TreeSaveNodeProperties";
import { TreeLoadNodePropertiesDialog, TreeLoadNodeProperties } from "../components/dialogs/TreeLoadNodeProperties";
import { TreeFindNodePropertiesDialog, TreeFindNodeProperties } from "../components/dialogs/TreeFindNodeProperties";
import { TreeSearchNodePropertiesDialog, TreeSearchNodeProperties } from "../components/dialogs/TreeSearchNodeProperties";
import { TreeSaveSecureNodePropertiesDialog, TreeSaveSecureNodeProperties } from "../components/dialogs/TreeSaveSecureNodeProperties";
import { TreeLoadSecureNodePropertiesDialog, TreeLoadSecureNodeProperties } from "../components/dialogs/TreeLoadSecureNodeProperties";
import { TreeValidateSecureNodePropertiesDialog, TreeValidateSecureNodeProperties } from "../components/dialogs/TreeValidateSecureNodeProperties";
import { TreeGetMetadataNodePropertiesDialog, TreeGetMetadataNodeProperties } from "../components/dialogs/TreeGetMetadataNodeProperties";
import { TreeWalkNodePropertiesDialog, TreeWalkNodeProperties } from "../components/dialogs/TreeWalkNodeProperties";
import { TraverseNodePropertiesDialog, TraverseNodeProperties } from "../components/dialogs/TraverseNodeProperties";
import { TreeToXMLNodePropertiesDialog, TreeToXMLNodeProperties } from "../components/dialogs/TreeToXMLNodeProperties";
import { TreeToYAMLNodePropertiesDialog, TreeToYAMLNodeProperties } from "../components/dialogs/TreeToYAMLNodeProperties";
import { XMLNodePropertiesDialog, XMLNodeProperties } from "../components/dialogs/XMLNodeProperties";
import { YAMLNodePropertiesDialog, YAMLNodeProperties } from "../components/dialogs/YAMLNodeProperties";
import { IfNodePropertiesDialog, IfNodeProperties } from "../components/dialogs/IfNodeProperties";
import { IifNodePropertiesDialog, IifNodeProperties } from "../components/dialogs/IifNodeProperties";
import { WhileNodePropertiesDialog, WhileNodeProperties } from "../components/dialogs/WhileNodeProperties";
import { CBQueryNodePropertiesDialog, CBQueryNodeProperties } from "../components/dialogs/CBQueryNodeProperties";
import { ThemeToggle } from "../components/ui/ThemeToggle";
import { DirectionToggle } from "../components/ui/DirectionToggle";
import { NestingToggle } from "../components/NestingToggle";
import { useTheme } from "../contexts/ThemeContext";
import { useFlowControl } from "../contexts/FlowControlContext";
import { useNesting } from "../contexts/NestingContext";
import type { Subflow } from "../contexts/NestingContext";
import { LogiconData } from "../data/logicons";

// Diagram persistence types
interface DiagramData {
  name: string;
  nodes: Node[];
  edges: Edge[];
  nestingRelations: any[];
  subflows?: Record<string, Subflow>;
  groupCount: number;
  created: string;
  modified: string;
}

const initialNodes: Node[] = [
  {
    id: "start",
    type: "logicon",
    position: { x: 50, y: 50 },
    data: { 
      label: "Start", 
      icon: "🚀",
      category: "control"
    },
  },
];

const initialEdges: Edge[] = [];

// Define nodeTypes outside component to prevent React Flow warnings
const nodeTypes = {
  logicon: LogiconNode,
  group: GroupNode,
};

export default function VisualDSLPrototype() {
  const { theme } = useTheme();
  const { direction, selectedNodeId, setSelectedNodeId } = useFlowControl();
  const { nestingMode, selectedParentId, setSelectedParentId, nestingRelations, addNestingRelation, getChildrenOf, removeNestingRelation, setNestingMode, wrapGroupAsSubflow, isSubflow, getSubflow, getAllSubflows, replaceAllSubflows } = useNesting();
  const [nodes, setNodes] = React.useState<Node[]>(initialNodes);
  const [edges, setEdges] = React.useState<Edge[]>(initialEdges);
  
  // Simple group counter - increment when nesting is created
  const [groupCount, setGroupCount] = React.useState(0);
  
  // Diagram state
  const [currentDiagramName, setCurrentDiagramName] = React.useState('Untitled Diagram');
  
  // Counter for unique node IDs
  const nodeCounterRef = React.useRef(0);
  
  // Context menu state
  const [contextMenu, setContextMenu] = React.useState<{
    x: number;
    y: number;
    nodeId: string;
    nodeLabel: string;
    isGroupRoot: boolean;
  } | null>(null);

  // Properties dialog state
  const [propertiesDialog, setPropertiesDialog] = React.useState<{
    nodeId: string;
    nodeType: string;
    properties: any;
  } | null>(null);

  // No auto-subflow wrapping; manual only

  // Diagram persistence functions
  const createDiagramData = (): DiagramData => {
    return {
      name: currentDiagramName,
      nodes,
      edges,
      nestingRelations,
      subflows: getAllSubflows(),
      groupCount,
      created: new Date().toISOString(),
      modified: new Date().toISOString()
    };
  };

  const saveDiagram = () => {
    const diagramData = createDiagramData();
    const key = `diagram_${currentDiagramName.replace(/[^a-zA-Z0-9]/g, '_')}`;
    localStorage.setItem(key, JSON.stringify(diagramData));
    
    // Save to list of diagrams
    const existingDiagrams = JSON.parse(localStorage.getItem('diagram_list') || '[]');
    if (!existingDiagrams.includes(key)) {
      existingDiagrams.push(key);
      localStorage.setItem('diagram_list', JSON.stringify(existingDiagrams));
    }
    
    alert(`Diagram "${currentDiagramName}" saved successfully!`);
  };

  const loadDiagram = (jsonData: string) => {
    try {
      // Sanitize input and handle potential double-encoding or BOMs
      let raw = typeof jsonData === 'string' ? jsonData : JSON.stringify(jsonData);
      raw = raw.trim().replace(/^\uFEFF/, ''); // strip BOM if present

      let parsed: any = JSON.parse(raw);
      // If the first parse yields a string that looks like JSON, parse again
      if (typeof parsed === 'string') {
        const inner = parsed.trim();
        if ((inner.startsWith('{') && inner.endsWith('}')) || (inner.startsWith('[') && inner.endsWith(']'))) {
          parsed = JSON.parse(inner);
        }
      }

      const diagramData: DiagramData = parsed;
      
      // Clear existing nesting relations
      nestingRelations.forEach(rel => {
        removeNestingRelation(rel.parentId, rel.childId);
      });
      
      // Load the diagram data with deduplication
      const originalNodes = diagramData.nodes || [];
      const originalEdges = diagramData.edges || [];
      
      const uniqueNodes = originalNodes.filter((node, index, array) => {
        return array.findIndex((n: Node) => n.id === node.id) === index;
      });
      const uniqueEdges = originalEdges.filter((edge, index, array) => {
        return array.findIndex((e: Edge) => e.id === edge.id) === index;
      });
      
      // Log warnings if duplicates were found
      if (uniqueNodes.length !== originalNodes.length) {
        console.warn(`Removed ${originalNodes.length - uniqueNodes.length} duplicate nodes during diagram load`);
      }
      if (uniqueEdges.length !== originalEdges.length) {
        console.warn(`Removed ${originalEdges.length - uniqueEdges.length} duplicate edges during diagram load`);
      }
      
  setNodes(uniqueNodes);
      setEdges(uniqueEdges);
      setGroupCount(diagramData.groupCount || 0);
      
      // Sync diagram name - prefer Start node name if available, otherwise use saved name
      const startNode = (diagramData.nodes || []).find(node => node.data.label === 'Start');
      if (startNode && startNode.data.properties && startNode.data.properties.name) {
        setCurrentDiagramName(startNode.data.properties.name);
      } else {
        setCurrentDiagramName(diagramData.name);
      }
      
      // Restore nesting relations
      if (diagramData.nestingRelations) {
        diagramData.nestingRelations.forEach(rel => {
          addNestingRelation(rel);
        });
      }

      // Restore subflows (metadata like names/collapsed state)
      if (diagramData.subflows) {
        replaceAllSubflows(diagramData.subflows);
      } else {
        // If none saved, let auto-sync rebuild from nestingRelations
        replaceAllSubflows({});
      }
      
      // Clear selections
      setSelectedNodeId('');
      setSelectedParentId(null);
      setNestingMode(false);
      
      alert(`Diagram "${diagramData.name}" loaded successfully!`);
    } catch (error) {
      alert('Failed to load diagram. Invalid JSON format.');
      try {
        const preview = (typeof jsonData === 'string' ? jsonData : JSON.stringify(jsonData)).slice(0, 200);
        console.error('Load diagram error:', error, '\nPreview:', preview);
      } catch (_) {
        console.error('Load diagram error:', error);
      }
    }
  };

  const newDiagram = () => {
    if (nodes.length > 0 || edges.length > 0) {
      const confirmed = window.confirm('Create new diagram? This will clear the current work.');
      if (!confirmed) return;
    }
    
    // Clear everything
    setNodes([]);
    setEdges([]);
    setGroupCount(0);
    setCurrentDiagramName('Untitled Diagram');
    
    // Clear nesting relations
    nestingRelations.forEach(rel => {
      removeNestingRelation(rel.parentId, rel.childId);
    });
    
    // Clear selections
    setSelectedNodeId('');
    setSelectedParentId(null);
    setNestingMode(false);
    
    // Reset node counter
    nodeCounterRef.current = 0;
  };

  const exportDiagram = async () => {
    const diagramData = createDiagramData();
    const jsonString = JSON.stringify(diagramData, null, 2);
    
    // Try to use the modern File System Access API if available
    if ('showSaveFilePicker' in window) {
      try {
        const fileHandle = await (window as any).showSaveFilePicker({
          suggestedName: `${currentDiagramName}.json`,
          types: [
            {
              description: 'JSON files',
              accept: {
                'application/json': ['.json'],
              },
            },
          ],
        });
        
        const writable = await fileHandle.createWritable();
        await writable.write(jsonString);
        await writable.close();
        
        alert(`Diagram "${currentDiagramName}" exported successfully!`);
        return;
      } catch (error) {
        // User cancelled or API not supported, fall back to download
        console.log('File picker cancelled or not supported, falling back to download');
      }
    }
    
    // Fallback: traditional download method
    const blob = new Blob([jsonString], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    
    const link = document.createElement('a');
    link.href = url;
    link.download = `${currentDiagramName}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    
    alert(`Diagram "${currentDiagramName}" downloaded to your Downloads folder.\n\nTip: To save to the diagrams/ folder, manually move the file from Downloads to your project's diagrams/ directory.`);
  };

  // Subflow: wrap selected context as a subflow
  const wrapSelectedGroupAsSubflow = React.useCallback(() => {
    // Determine parent candidate from multiple contexts
    let candidateParentId: string | null = null;
    if (nestingMode && selectedParentId) {
      candidateParentId = selectedParentId;
    } else if (selectedNodeId) {
      if (selectedNodeId.startsWith('group-')) {
        candidateParentId = selectedNodeId.replace(/^group-/, '');
      } else if (nestingRelations.some(rel => rel.parentId === selectedNodeId)) {
        candidateParentId = selectedNodeId;
      } else {
        const parentRel = nestingRelations.find(rel => rel.childId === selectedNodeId);
        if (parentRel) candidateParentId = parentRel.parentId;
      }
    }
    if (!candidateParentId) {
      console.warn('Wrap as Subflow: no valid parent context found');
      return;
    }
    const groupId = `group-${candidateParentId}`;
    if (isSubflow(groupId)) return;

    // Ensure a group container node exists so the frame has bounds to anchor to
    const groupExists = nodes.some(n => n.id === groupId);
    if (!groupExists) {
      const kids = getChildrenOf(candidateParentId);
      if (kids.length > 0) {
        // create group based on current layout of parent+children
        createGroupForNesting(candidateParentId, kids[0].childId);
      }
    }

    console.log('Wrapping as subflow:', { parentId: candidateParentId, groupId });
    wrapGroupAsSubflow(groupId, { name: `${candidateParentId} Subflow` });
  }, [nestingMode, selectedParentId, selectedNodeId, nestingRelations, isSubflow, wrapGroupAsSubflow, nodes]);

  // Delete node and handle nesting group logic
  const deleteNode = (nodeId: string) => {
    console.log('Deleting node:', nodeId);
    
    // Check if this node is a nesting group root
    const isGroupRoot = nestingRelations.some(rel => rel.parentId === nodeId);
    
    if (isGroupRoot) {
      // Delete entire nesting group
      const children = getChildrenOf(nodeId);
      const allNodesToDelete = [nodeId, ...children.map(c => c.childId)];
      const groupId = `group-${nodeId}`;
      
      console.log('Deleting nesting group:', { root: nodeId, children: children.map(c => c.childId), groupNode: groupId });
      
      // Remove all nesting relations for this group
      children.forEach(child => {
        removeNestingRelation(nodeId, child.childId);
      });
      
      // Decrement group count when deleting a group
      setGroupCount(prev => Math.max(0, prev - 1));
      
      // Remove all nodes in the group (including the group container)
      setNodes(nds => nds.filter(n => !allNodesToDelete.includes(n.id) && n.id !== groupId));
      
      // Remove all edges connected to any of these nodes
      setEdges(eds => eds.filter(edge => 
        !allNodesToDelete.includes(edge.source) && 
        !allNodesToDelete.includes(edge.target)
      ));
    } else {
      // Delete single node
      console.log('Deleting single node:', nodeId);
      
      // If this node is a child in a nesting group, remove the nesting relation
      const parentRelation = nestingRelations.find(rel => rel.childId === nodeId);
      if (parentRelation) {
        removeNestingRelation(parentRelation.parentId, nodeId);
        
        // Check if this was the last child - if so, decrement group count
        const remainingChildren = getChildrenOf(parentRelation.parentId);
        if (remainingChildren.length === 1) { // Will be 0 after we remove this one
          setGroupCount(prev => Math.max(0, prev - 1));
        }
        
        // Update the group bounds after removing child
        setTimeout(() => {
          const remainingChildrenAfterRemoval = getChildrenOf(parentRelation.parentId);
          if (remainingChildrenAfterRemoval.length === 0) {
            // No more children, remove the group container
            const groupId = `group-${parentRelation.parentId}`;
            setNodes(nds => nds.filter(n => n.id !== groupId));
          } else {
            // Update group bounds
            updateGroupBounds(parentRelation.parentId);
          }
        }, 0);
      }
      
      // Remove the node
      setNodes(nds => nds.filter(n => n.id !== nodeId));
      
      // Remove all edges connected to this node (but don't delete connected nodes)
      setEdges(eds => eds.filter(edge => 
        edge.source !== nodeId && edge.target !== nodeId
      ));
    }
    
    // Clear selection if the deleted node was selected
    if (selectedNodeId === nodeId) {
      setSelectedNodeId('');
    }
    if (selectedParentId === nodeId) {
      setSelectedParentId(null);
      setNestingMode(false);
    }
    
    setContextMenu(null);
  };

  // Handle right-click on nodes
  const onNodeContextMenu = React.useCallback(
    (event: React.MouseEvent, node: Node) => {
      event.preventDefault();
      
      const isGroupRoot = nestingRelations.some(rel => rel.parentId === node.id);
      
      setContextMenu({
        x: event.clientX,
        y: event.clientY,
        nodeId: node.id,
        nodeLabel: node.data.label,
        isGroupRoot,
      });
    },
    [nestingRelations]
  );

  // Handle properties dialog
  const openPropertiesDialog = React.useCallback((nodeId: string) => {
    const node = nodes.find(n => n.id === nodeId);
    if (node) {
      console.log('Opening properties for node:', node); // Debug log
      
      // Determine node type based on label or category
      let nodeType = 'unknown';
      const label = node.data.label;
      const category = node.data.category;
      
      if ((label === 'Start' || label === 'start') && category === 'control') {
        nodeType = 'start';
      } else if ((label === 'Declare' || label === 'declare') && category === 'value') {
        nodeType = 'declare';
      } else if ((label === 'If' || label === 'if') && category === 'control') {
        nodeType = 'if';
      } else if ((label === 'Iif' || label === 'IIf'|| label === 'iif') && category === 'comparison') {
        nodeType = 'iif';
      } else if ((label === 'While' || label === 'while') && category === 'control') {
        nodeType = 'while';
      } else if ((label === 'CB Query' || label === 'cbQuery') && category === 'couchbase') {
        nodeType = 'cbQuery';
      } else if ((label === 'Add Child' || label === 'addChild') && category === 'node') {
        nodeType = 'addChild';
      } else if ((label === 'Remove Child' || label === 'removeChild') && category === 'node') {
        nodeType = 'removeChild';
      } else if ((label === 'Child Count' || label === 'childCount') && category === 'node') {
        nodeType = 'childCount';
      } else if ((label === 'Clear' || label === 'clear') && category === 'node') {
        nodeType = 'clear';
      } else if ((label === 'First Child' || label === 'firstChild') && category === 'node') {
        nodeType = 'firstChild';
      } else if ((label === 'Last Child' || label === 'lastChild') && category === 'node') {
        nodeType = 'lastChild';
      } else if ((label === 'CSV Node' || label === 'csvNode') && category === 'node') {
        nodeType = 'csvNode';
      } else if ((label === 'JSON Node' || label === 'jsonNode') && category === 'node') {
        nodeType = 'jsonNode';
      } else if ((label === 'XML Node' || label === 'xmlNode') && category === 'node') {
        nodeType = 'xmlNode';
      } else if ((label === 'YAML Node' || label === 'yamlNode') && category === 'node') {
        nodeType = 'yamlNode';
      } else if ((label === 'Map Node' || label === 'mapNode') && category === 'node') {
        nodeType = 'mapNode';
      } else if ((label === 'List' || label === 'list') && category === 'node') {
        nodeType = 'list';
      } else if ((label === 'Node To String' || label === 'nodeToString') && category === 'node') {
        nodeType = 'nodeToString';
      } else if ((label === 'Find By Name' || label === 'findByName') && category === 'node') {
        nodeType = 'findByName';
      } else if ((label === 'Traverse Node' || label === 'traverseNode') && category === 'node') {
        nodeType = 'traverseNode';
      } else if ((label === 'Query Node' || label === 'queryNode') && category === 'node') {
        nodeType = 'queryNode';
      } else if ((label === 'Get Attribute' || label === 'getAttribute') && category === 'node') {
        nodeType = 'getAttribute';
      } else if ((label === 'Set Attribute' || label === 'setAttribute') && category === 'node') {
        nodeType = 'setAttribute';
      } else if ((label === 'Set Attributes' || label === 'setAttributes') && category === 'node') {
        nodeType = 'setAttributes';
      } else if ((label === 'Remove Attribute' || label === 'removeAttribute') && category === 'node') {
        nodeType = 'removeAttribute';
      } else if ((label === 'Get Child At' || label === 'getChildAt') && category === 'node') {
        nodeType = 'getChildAt';
      } else if ((label === 'Get Child By Name' || label === 'getChildByName') && category === 'node') {
        nodeType = 'getChildByName';
      } else if ((label === 'Get Depth' || label === 'getDepth') && category === 'node') {
        nodeType = 'getDepth';
      } else if ((label === 'Get Level' || label === 'getLevel') && category === 'node') {
        nodeType = 'getLevel';
      } else if ((label === 'Get Name' || label === 'getName') && category === 'node') {
        nodeType = 'getName';
      } else if ((label === 'Set Name' || label === 'setName') && category === 'node') {
        nodeType = 'setName';
      } else if ((label === 'Get Parent' || label === 'getParent') && category === 'node') {
        nodeType = 'getParent';
      } else if ((label === 'Get Path' || label === 'getPath') && category === 'node') {
        nodeType = 'getPath';
      } else if ((label === 'Get Root' || label === 'getRoot') && category === 'node') {
        nodeType = 'getRoot';
      } else if ((label === 'Get Siblings' || label === 'getSiblings') && category === 'node') {
        nodeType = 'getSiblings';
      } else if ((label === 'Get Text' || label === 'getText') && category === 'node') {
        nodeType = 'getText';
      } else if ((label === 'Set Text' || label === 'setText') && category === 'node') {
        nodeType = 'setText';
      } else if ((label === 'Has Attribute' || label === 'hasAttribute') && category === 'node') {
        nodeType = 'hasAttribute';
      } else if ((label === 'Is Leaf' || label === 'isLeaf') && category === 'node') {
        nodeType = 'isLeaf';
      } else if ((label === 'Is Root' || label === 'isRoot') && category === 'node') {
        nodeType = 'isRoot';
      } else if ((label === 'Tree Save' || label === 'treeSave') && category === 'tree') {
        nodeType = 'treeSave';
      } else if ((label === 'Tree Load' || label === 'treeLoad') && category === 'tree') {
        nodeType = 'treeLoad';
      } else if ((label === 'Tree Find' || label === 'treeFind') && category === 'tree') {
        nodeType = 'treeFind';
      } else if ((label === 'Tree Search' || label === 'treeSearch') && category === 'tree') {
        nodeType = 'treeSearch';
      } else if ((label === 'Tree Save Secure' || label === 'treeSaveSecure') && category === 'tree') {
        nodeType = 'treeSaveSecure';
      } else if ((label === 'Tree Load Secure' || label === 'treeLoadSecure') && category === 'tree') {
        nodeType = 'treeLoadSecure';
      } else if ((label === 'Tree Validate Secure' || label === 'treeValidateSecure') && category === 'tree') {
        nodeType = 'treeValidateSecure';
      } else if ((label === 'Tree Get Metadata' || label === 'treeGetMetadata') && category === 'tree') {
        nodeType = 'treeGetMetadata';
      } else if ((label === 'Tree To XML' || label === 'treeToXML') && category === 'tree') {
        nodeType = 'treeToXML';
      } else if ((label === 'Tree To YAML' || label === 'treeToYAML') && category === 'tree') {
        nodeType = 'treeToYAML';
      } else if ((label === 'Tree Walk' || label === 'treeWalk') && category === 'tree') {
        nodeType = 'treeWalk';
      } else if ((label === 'Add To' || label === 'addTo') && category === 'array') {
        nodeType = 'addTo';
      } else if ((label === 'Log Print' || label === 'LogPrint' || label === 'logPrint') && category === 'system') {
        nodeType = 'logPrint';
      } else if ((label === 'Create Transform' || label === 'CreateTransform' || label === 'createTransform') && category === 'etl') {
        nodeType = 'createTransform';
      } else if ((label === 'Add Mapping' || label === 'AddMapping' || label === 'addMapping') && category === 'etl') {
        nodeType = 'addMapping';
      } else if (label === 'New Tree' && category === 'tree') {
        nodeType = 'newTree';
      } else if (label === 'Create' && (category === 'node' || category === 'tree')) {
        nodeType = 'create';
      } else if ((label === 'Parse JSON' || category === 'parseJSON' || category === 'json')) {
        nodeType = 'parseJSON';
      } else if ((label === 'Array' || label === 'array') && category === 'array') {
        nodeType = 'array';
      }
      
      console.log(`Node type determined: ${nodeType} for label: "${label}", category: "${category}"`); // Debug log
      
      setPropertiesDialog({
        nodeId,
        nodeType,
        properties: node.data.properties || {}
      });
    }
    setContextMenu(null);
  }, [nodes]);

  const saveNodeProperties = React.useCallback((nodeId: string, properties: any) => {
    setNodes((nodes) =>
      nodes.map((node) => {
        if (node.id === nodeId) {
          const updatedNode = { ...node, data: { ...node.data, properties } };
          // If this is a Start node, sync the diagram name with the Start node's name
          if (node.data.label === 'Start' && (properties as any).name) {
            setCurrentDiagramName((properties as any).name);
          }
          return updatedNode;
        }
        return node;
      })
    );
  }, []);

  // Function to sync Start node name when diagram name changes
  const updateDiagramName = React.useCallback((newName: string) => {
    setCurrentDiagramName(newName);
    
    // Find Start node and update its name property
    setNodes((nodes) =>
      nodes.map((node) => {
        if (node.data.label === 'Start') {
          return {
            ...node,
            data: {
              ...node.data,
              properties: {
                ...node.data.properties,
                name: newName
              }
            }
          };
        }
        return node;
      })
    );
  }, []);

  const onNodesChange = React.useCallback(
    (changes: NodeChange[]) => {
      setNodes((prev) => {
        // First, apply the default changes
        let next = applyNodeChanges(changes, prev);
        // Then, if a group node moved, move its parent and children by the same delta
        const positionChanges = changes.filter((c: any) => c.type === 'position' && typeof c.id === 'string' && c.id.startsWith('group-')) as any[];
        for (const ch of positionChanges) {
          const groupId = ch.id as string;
          const parentId = groupId.replace(/^group-/, '');
          const prevGroup = prev.find(n => n.id === groupId);
          const newGroup = next.find(n => n.id === groupId);
          if (!prevGroup || !newGroup) continue;
          const dx = (newGroup.position.x - prevGroup.position.x);
          const dy = (newGroup.position.y - prevGroup.position.y);
          if (dx === 0 && dy === 0) continue;
          const children = getChildrenOf(parentId).map(c => c.childId);
          const affectedIds = new Set<string>([parentId, ...children]);
          next = next.map(n => {
            if (affectedIds.has(n.id)) {
              return { ...n, position: { x: n.position.x + dx, y: n.position.y + dy } };
            }
            return n;
          });
        }

        // Also: if a parent or child in a subflow moved, treat it as dragging the entire subflow
        const nodePositionChanges = changes.filter((c: any) => c.type === 'position' && typeof c.id === 'string' && !String(c.id).startsWith('group-')) as any[];
        for (const ch of nodePositionChanges) {
          const movedId = ch.id as string;
          // Determine if this node is a parent or a child in a nesting group
          let parentId: string | null = null;
          const isParent = getChildrenOf(movedId).length > 0;
          if (isParent) {
            parentId = movedId;
          } else {
            const rel = nestingRelations.find(r => r.childId === movedId);
            parentId = rel ? rel.parentId : null;
          }
          if (!parentId) continue;
          const groupId = `group-${parentId}`;
          const prevNode = prev.find(n => n.id === movedId);
          const nextNode = next.find(n => n.id === movedId);
          if (!prevNode || !nextNode) continue;
          const dx = (nextNode.position.x - prevNode.position.x);
          const dy = (nextNode.position.y - prevNode.position.y);
          if (dx === 0 && dy === 0) continue;

          const children = getChildrenOf(parentId).map(c => c.childId);
          const affectedIds = new Set<string>([parentId, ...children]);
          next = next.map(n => {
            if (n.id === groupId) {
              return { ...n, position: { x: n.position.x + dx, y: n.position.y + dy } };
            }
            if (affectedIds.has(n.id) && n.id !== movedId) {
              return { ...n, position: { x: n.position.x + dx, y: n.position.y + dy } };
            }
            return n;
          });
        }
        return next;
      });
    },
    [getChildrenOf, nestingRelations]
  );
  const onEdgesChange = React.useCallback(
    (changes: EdgeChange[]) => setEdges((eds) => applyEdgeChanges(changes, eds)),
    []
  );
  const onConnect = React.useCallback(
    (connection: Connection) => setEdges((eds) => addEdge(connection, eds)),
    []
  );

  // Handle edge updates (moving connectors between handles)
  const onEdgeUpdate = React.useCallback(
    (oldEdge: Edge, newConnection: Connection) => {
      setEdges((eds) => reconnectEdge(oldEdge, newConnection, eds));
    },
    []
  );

  // Force create groups for all existing nesting relationships (debugging helper)
  const createMissingGroups = () => {
    console.log('Creating missing groups for existing relationships...');
    const parentIds = [...new Set(nestingRelations.map(rel => rel.parentId))];
    
    parentIds.forEach(parentId => {
      const groupId = `group-${parentId}`;
      const groupExists = nodes.some(n => n.id === groupId);
      
      if (!groupExists) {
        console.log(`Missing group for parent ${parentId}, creating...`);
        const children = getChildrenOf(parentId);
        if (children.length > 0) {
          const firstChild = children[0];
          createGroupForNesting(parentId, firstChild.childId);
        }
      }
    });
  };

  // Create a group node that contains nested function calls (used by Fix Groups button)
  const createGroupForNesting = (parentId: string, childId: string) => {
    setNodes(currentNodes => {
      const parentNode = currentNodes.find(n => n.id === parentId);
      const childNode = currentNodes.find(n => n.id === childId);
      
      if (!parentNode || !childNode) {
        return currentNodes;
      }
      
      // Check if there's already a group for this parent
      const existingGroupId = `group-${parentId}`;
      const existingGroup = currentNodes.find(n => n.id === existingGroupId);
      
      if (existingGroup) {
        // Update existing group bounds
        const groupBounds = calculateGroupBoundsForNodes(currentNodes, parentId);
        if (!groupBounds) return currentNodes;
        
        const padding = 15; // exact desired margin
        return currentNodes.map(node => {
          if (node.id === existingGroupId) {
            const containerWidth = groupBounds.width + (padding * 2);
            const containerHeight = groupBounds.height + (padding * 2);
            return {
              ...node,
              position: { x: (groupBounds.minX - padding), y: (groupBounds.minY - padding) },
              style: {
                width: containerWidth,
                height: containerHeight,
              }
            };
          }
          return node;
        });
      } else {
        // Create new group
        const groupBounds = calculateGroupBoundsForNodes(currentNodes, parentId);
        if (!groupBounds) return currentNodes;
        
        const padding = 15;
        const groupId = `group-${parentId}`;
        const groupNode: Node = {
          id: groupId,
          type: 'group',
          position: { x: (groupBounds.minX - padding), y: (groupBounds.minY - padding) },
          style: {
            width: (groupBounds.width + (padding * 2)),
            height: groupBounds.height + (padding * 2),
          },
          data: {
            label: `${parentNode.data.label}(...)`,
          },
          zIndex: 0,
          draggable: true,
        };
        
        return [...currentNodes, groupNode];
      }
    });
  };

  // Update group bounds when nesting relationships change
  const updateGroupBounds = (parentId: string) => {
    const groupBounds = calculateGroupBounds(parentId);
    if (!groupBounds) return;
    
    const padding = 15;
    const groupId = `group-${parentId}`;
    
    setNodes(nds => nds.map(node => {
      if (node.id === groupId) {
        const containerWidth = groupBounds.width + (padding * 2);
        const containerHeight2 = groupBounds.height + (padding * 2);
        return {
          ...node,
          position: { x: (groupBounds.minX - padding), y: (groupBounds.minY - padding) },
          style: {
            width: containerWidth,
            height: containerHeight2,
          }
        };
      }
      return node;
    }));
  };

  const onNodeClick = React.useCallback(
    (event: React.MouseEvent, node: Node) => {
      console.log('Node clicked:', { nodeId: node.id, nestingMode, selectedParentId });
      
      // In nesting mode, allow changing the selected parent to support multi-level nesting
      if (nestingMode) {
        // Only allow selecting logicon nodes (not group nodes) as parents
        if (node.type === 'logicon') {
          console.log('Changing parent selection in nesting mode to:', node.id);
          setSelectedParentId(node.id);
        }
        return;
      }
      
      // Normal mode - handle group selection behavior
      let nodeToSelect = node.id;
      // If clicking on a group container, select the underlying parent node instead
      if (node.type === 'group' && node.id.startsWith('group-')) {
        nodeToSelect = node.id.replace(/^group-/, '');
      }
      
      // If clicking on a child node in a nesting group, select the parent instead
      const parentRelation = nestingRelations.find(rel => rel.childId === node.id);
      if (parentRelation) {
        nodeToSelect = parentRelation.parentId;
      }
      
      setSelectedNodeId(nodeToSelect);
    },
    [nestingMode, selectedParentId, setSelectedNodeId, setSelectedParentId, nestingRelations]
  );

  // Find the last node in the logical flow (rightmost and bottommost)
  const findLastNode = () => {
    if (nodes.length === 0) return null;
    return nodes.reduce((lastNode, currentNode) => {
      if (!lastNode) return currentNode;
      // Prioritize nodes that are further down and to the right
      if (currentNode.position.y > lastNode.position.y || 
          (currentNode.position.y === lastNode.position.y && currentNode.position.x > lastNode.position.x)) {
        return currentNode;
      }
      return lastNode;
    });
  };

  // Calculate position based on direction and reference node
  const calculateNewPosition = (referenceNode: Node) => {
    const spacing = 150; // Distance between nodes
    
    switch (direction) {
      case 'down':
        return {
          x: referenceNode.position.x,
          y: referenceNode.position.y + spacing
        };
      case 'right':
        return {
          x: referenceNode.position.x + spacing,
          y: referenceNode.position.y
        };
      case 'up':
        return {
          x: referenceNode.position.x,
          y: referenceNode.position.y - spacing
        };
      default:
        return {
          x: referenceNode.position.x,
          y: referenceNode.position.y + spacing
        };
    }
  };

  // Regular add logicon with random placement (now the modifier behavior)
  const addLogiconRandom = (logicon: LogiconData, withModifier: boolean = false) => {
    const id = `${logicon.id}-${Date.now()}-${++nodeCounterRef.current}`;
    
    // Set default properties based on node type
    let defaultProperties = {};
    if (logicon.label === 'Start') {
      defaultProperties = { name: 'NameOfRule' };
      // If diagram is still untitled, set it to the Start node's default name
      if (currentDiagramName === 'Untitled Diagram') {
        setCurrentDiagramName('NameOfRule');
      }
    } else if (logicon.label === 'Declare') {
      defaultProperties = {
        isGlobal: false,
        variableName: 'myVar',
        typeSpecifier: 'S'
      };
    } else if (logicon.label === 'Create') {
      defaultProperties = {
        nodeName: 'MyNode'
      };
    } else if (logicon.label === 'Parse JSON') {
      defaultProperties = {
        jsonString: '{"key": "value"}',
        nodeName: 'root'
      };
    }
    
    const newNode: Node = {
      id,
      type: "logicon",
      position: {
        x: Math.random() * 600 + 100,
        y: Math.random() * 400 + 100,
      },
      data: { 
        label: logicon.label,
        icon: logicon.icon,
        category: logicon.category,
        properties: defaultProperties
      },
    };
    setNodes((nds) => [...nds, newNode]);
  };

  // Calculate the bounds of a nesting group
  const calculateGroupBounds = (parentId: string) => {
    const children = getChildrenOf(parentId);
    const parentNode = nodes.find(n => n.id === parentId);
    
    if (!parentNode || children.length === 0) {
      return null;
    }
    
    // Get all nodes in the group (parent + children)
    const groupNodeIds = [parentId, ...children.map(c => c.childId)];
    const groupNodes = nodes.filter(n => groupNodeIds.includes(n.id));
    
    if (groupNodes.length === 0) return null;
    
    // Calculate bounds using measured node sizes when available
  const DEFAULT_W = 140;
  const DEFAULT_H = 90;
    const minX = Math.min(...groupNodes.map(n => n.position.x));
    const maxX = Math.max(
      ...groupNodes.map(n => n.position.x + (typeof (n as any).width === 'number' ? (n as any).width : DEFAULT_W))
    );
    const minY = Math.min(...groupNodes.map(n => n.position.y));
    const maxY = Math.max(
      ...groupNodes.map(n => n.position.y + (typeof (n as any).height === 'number' ? (n as any).height : DEFAULT_H))
    );
    
    return {
      minX,
      maxX,
      minY,
      maxY,
      centerX: (minX + maxX) / 2,
      centerY: (minY + maxY) / 2,
      width: maxX - minX,
      height: maxY - minY,
    };
  };

  // Version that accepts nodes array for state consistency
  const calculateGroupBoundsForNodes = (nodeArray: Node[], parentId: string) => {
    const children = getChildrenOf(parentId);
    const parentNode = nodeArray.find(n => n.id === parentId);
    
    if (!parentNode || children.length === 0) {
      return null;
    }
    
    // Get all nodes in the group (parent + children)
    const groupNodeIds = [parentId, ...children.map(c => c.childId)];
    const groupNodes = nodeArray.filter(n => groupNodeIds.includes(n.id));
    
    if (groupNodes.length === 0) {
      return null;
    }
    
    // Calculate bounds using measured node sizes when available
  const DEFAULT_W = 140;
  const DEFAULT_H = 90;
    const minX = Math.min(...groupNodes.map(n => n.position.x));
    const maxX = Math.max(
      ...groupNodes.map(n => n.position.x + (typeof (n as any).width === 'number' ? (n as any).width : DEFAULT_W))
    );
    const minY = Math.min(...groupNodes.map(n => n.position.y));
    const maxY = Math.max(
      ...groupNodes.map(n => n.position.y + (typeof (n as any).height === 'number' ? (n as any).height : DEFAULT_H))
    );
    
    return {
      minX,
      maxX,
      minY,
      maxY,
      centerX: (minX + maxX) / 2,
      centerY: (minY + maxY) / 2,
      width: maxX - minX,
      height: maxY - minY
    };
  };

  // Add logicon in flow (now the default single-click behavior)
  const addLogiconFlow = (logicon: LogiconData) => {
    const id = `${logicon.id}-${Date.now()}-${++nodeCounterRef.current}`;
    
    // In nesting mode, if we have a selected parent, add as child
    if (nestingMode && selectedParentId) {
      console.log('Adding child node in nesting mode:', { parent: selectedParentId, child: id });
      
      // Find the parent node
      const parentNode = nodes.find(n => n.id === selectedParentId);
      if (!parentNode) return;
      
      // Calculate child position based on direction and existing children count
      const childCount = getChildrenOf(selectedParentId).length;
      const spacing = 150;
      let childPosition = { x: 0, y: 0 };
      
      switch (direction) {
        case 'down':
          childPosition = {
            x: parentNode.position.x,
            y: parentNode.position.y + spacing + (childCount * 100)
          };
          break;
        case 'right':
          childPosition = {
            x: parentNode.position.x + spacing + (childCount * 150),
            y: parentNode.position.y
          };
          break;
        case 'up':
          childPosition = {
            x: parentNode.position.x,
            y: parentNode.position.y - spacing - (childCount * 100)
          };
          break;
        default: // down
          childPosition = {
            x: parentNode.position.x,
            y: parentNode.position.y + spacing + (childCount * 100)
          };
      }
      
      // Set default properties based on node type
      let defaultProperties = {};
      if (logicon.label === 'Start') {
        defaultProperties = { name: 'NameOfRule' };
        // If diagram is still untitled, set it to the Start node's default name
        if (currentDiagramName === 'Untitled Diagram') {
          setCurrentDiagramName('NameOfRule');
        }
      } else if (logicon.label === 'Declare') {
        defaultProperties = {
          isGlobal: false,
          variableName: 'myVar',
          typeSpecifier: 'S'
        };
      } else if (logicon.label === 'Create') {
        defaultProperties = {
          nodeName: 'MyNode'
        };
      } else if (logicon.label === 'Parse JSON') {
        defaultProperties = {
          jsonString: '{"key": "value"}',
          nodeName: 'root'
        };
      }
      
      const newNode: Node = {
        id,
        type: "logicon",
        position: childPosition,
        data: { 
          label: logicon.label,
          icon: logicon.icon,
          category: logicon.category,
          properties: defaultProperties
        },
      };
      
      // Add the nesting relationship first
      const children = getChildrenOf(selectedParentId);
      const isFirstChild = children.length === 0; // Check if this is the first child
      
      addNestingRelation({
        parentId: selectedParentId,
        childId: id,
        order: children.length
      });
      
      // Increment group count when creating the first nesting relationship for a parent
      if (isFirstChild) {
        setGroupCount(prev => prev + 1);
      }
      
      // Create edge between parent and child with direction-appropriate handles
      let sourceHandle: string;
      let targetHandle: string;
      
      switch (direction) {
        case 'down':
          sourceHandle = 'bottom';
          targetHandle = 'top';
          break;
        case 'right':
          sourceHandle = 'right';
          targetHandle = 'left';
          break;
        case 'up':
          sourceHandle = 'top';
          targetHandle = 'bottom';
          break;
        default:
          sourceHandle = 'bottom';
          targetHandle = 'top';
      }
      
      const newEdge: Edge = {
        id: `${selectedParentId}-${id}`,
        source: selectedParentId,
        target: id,
        sourceHandle,
        targetHandle,
        type: 'default',
        style: { stroke: '#8b5cf6', strokeWidth: 2 }, // Purple to match nesting theme
        animated: false,
        updatable: true, // Allow moving connector endpoints
      };
      
      // Simple approach - just add the nodes and edges, visual grouping handled by CSS
      setNodes((nds) => [...nds, newNode]);
      setEdges((eds) => [...eds, newEdge]);
      // Auto-create/update the group container for subflow visuals
      if (isFirstChild) {
        setTimeout(() => createGroupForNesting(selectedParentId, id), 0);
      } else {
        setTimeout(() => updateGroupBounds(selectedParentId), 0);
      }
      
      return;
    }
    
    // Regular flow logic (when not in nesting mode)
    // Determine reference node (selected node or last node in flow)
    let referenceNode: Node | null = null;
    if (selectedNodeId) {
      referenceNode = nodes.find(node => node.id === selectedNodeId) || null;
      
      // If the selected node is part of a nesting group, find the root of that group
      if (referenceNode && !nestingMode) {
        // If a group container is selected, use its parent logicon as the reference
        if (referenceNode.type === 'group' && referenceNode.id.startsWith('group-')) {
          const parentId = referenceNode.id.replace(/^group-/, '');
          const parentNode = nodes.find(n => n.id === parentId) || null;
          if (parentNode) {
            referenceNode = parentNode;
          }
        }
        const parentRelation = nestingRelations.find(rel => rel.childId === selectedNodeId);
        if (parentRelation) {
          // This node is a child in a nesting - use the parent as reference
          referenceNode = nodes.find(node => node.id === parentRelation.parentId) || referenceNode;
        }
        // If it's already a parent node, we use it as-is
      }
    }
    if (!referenceNode) {
      referenceNode = findLastNode();
    }
    
    // If no reference node, fall back to random placement
    if (!referenceNode) {
      addLogiconRandom(logicon);
      return;
    }

    // Calculate position - if reference node is part of a nesting group, 
    // position relative to the group bounds
    let newPosition = calculateNewPosition(referenceNode);
    
    // Check if reference node has children (is part of a nesting group)
    const children = getChildrenOf(referenceNode.id);
    if (children.length > 0) {
      // Adjust position to account for the entire group
      const groupBounds = calculateGroupBounds(referenceNode.id);
      if (groupBounds) {
        switch (direction) {
          case 'down':
            newPosition = {
              x: groupBounds.centerX,
              y: groupBounds.maxY + 150
            };
            break;
          case 'right':
            newPosition = {
              // place outside the group plus the group padding (15)
              x: groupBounds.maxX + 150 + 15,
              // keep the same vertical placement as the selected/reference node
              y: referenceNode.position.y
            };
            break;
          case 'up':
            newPosition = {
              x: groupBounds.centerX,
              y: groupBounds.minY - 150
            };
            break;
          default:
            newPosition = {
              x: groupBounds.centerX,
              y: groupBounds.maxY + 150
            };
        }
      }
    }

    // Set default properties based on node type
    let defaultProperties = {};
    if (logicon.label === 'Start') {
      defaultProperties = { name: 'NameOfRule' };
      // If diagram is still untitled, set it to the Start node's default name
      if (currentDiagramName === 'Untitled Diagram') {
        setCurrentDiagramName('NameOfRule');
      }
    } else if (logicon.label === 'Declare') {
      defaultProperties = {
        isGlobal: false,
        variableName: 'myVar',
        typeSpecifier: 'S'
      };
    } else if (logicon.label === 'Create') {
      defaultProperties = {
        nodeName: 'MyNode'
      };
    } else if (logicon.label === 'Parse JSON') {
      defaultProperties = {
        jsonString: '{"key": "value"}',
        nodeName: 'root'
      };
    }

    const newNode: Node = {
      id,
      type: "logicon",
      position: newPosition,
      data: { 
        label: logicon.label,
        icon: logicon.icon,
        category: logicon.category,
        properties: defaultProperties
      },
    };

    // Create connection with direction-specific handles
    let sourceHandle: string;
    let targetHandle: string;
    
    switch (direction) {
      case 'down':
        sourceHandle = 'bottom';
        targetHandle = 'top';
        break;
      case 'right':
        sourceHandle = 'right';
        targetHandle = 'left';
        break;
      case 'up':
        sourceHandle = 'top';
        targetHandle = 'bottom';
        break;
      default:
        sourceHandle = 'bottom';
        targetHandle = 'top';
    }

  // If reference was a group, we already mapped to its parent above; use that id
  const parentForEdge = referenceNode.id;
    const newEdge: Edge = {
      id: `${parentForEdge}-${id}`,
      source: parentForEdge,
      target: id,
      sourceHandle,
      targetHandle,
      type: 'default',
      updatable: true, // Allow moving connector endpoints
    };

    setNodes((nds) => [...nds, newNode]);
    setEdges((eds) => [...eds, newEdge]);
    setSelectedNodeId(id); // Select the newly created node
  };

  return (
    <ReactFlowProvider>
      <div className={`flex flex-col h-screen ${theme === 'dark' ? 'dark' : ''}`}>
        {/* Top Toolbar */}
        <DiagramToolbar
          currentDiagramName={currentDiagramName}
          onDiagramNameChange={updateDiagramName}
          onNew={newDiagram}
          onSave={saveDiagram}
          onLoad={loadDiagram}
          onExport={exportDiagram}
          diagramData={createDiagramData()}
        />
        
        {/* Main Content */}
        <div className="flex flex-1 overflow-hidden">
          {/* Sidebar - Restore dedicated scrolling */}
          <div className="w-56 bg-white dark:bg-gray-900 shadow-lg border-r border-gray-200 dark:border-gray-700 p-4 space-y-4 flex flex-col overflow-hidden">
          <div className="flex items-center justify-between mb-4 flex-shrink-0">
            <h2 className="text-lg font-semibold text-gray-800 dark:text-gray-200">Logicons</h2>
            <ThemeToggle />
          </div>
          
          <div className="flex-shrink-0">
            <DirectionToggle />
          </div>
          
          <div className="flex-shrink-0">
            <NestingToggle />
          </div>

          {/* Subflow actions */}
          <div className="flex-shrink-0">
            <button
              onClick={wrapSelectedGroupAsSubflow}
              className="w-full mt-2 h-8 text-xs px-3 rounded bg-indigo-100 hover:bg-indigo-200 dark:bg-indigo-700 dark:hover:bg-indigo-600 text-indigo-800 dark:text-indigo-200 border border-indigo-300 dark:border-indigo-600"
              title="Wrap selected group as Subflow"
              disabled={(() => {
                let candidateParentId: string | null = null;
                if (nestingMode && selectedParentId) candidateParentId = selectedParentId;
                else if (selectedNodeId) {
                  if (selectedNodeId.startsWith('group-')) candidateParentId = selectedNodeId.replace(/^group-/, '');
                  else if (nestingRelations.some(rel => rel.parentId === selectedNodeId)) candidateParentId = selectedNodeId;
                  else {
                    const parentRel = nestingRelations.find(rel => rel.childId === selectedNodeId);
                    if (parentRel) candidateParentId = parentRel.parentId;
                  }
                }
                if (!candidateParentId) return true;
                return isSubflow(`group-${candidateParentId}`);
              })()}
            >
              🧩 Wrap as Subflow
            </button>
          </div>
          
          {/* Debug info */}
          <div className="text-xs text-gray-500 dark:text-gray-400 p-2 bg-gray-100 dark:bg-gray-800 rounded flex-shrink-0">
            <div>Nodes: {nodes.length}</div>
            <div>Groups: {groupCount}</div>
            <div>Relations: {nestingRelations.length}</div>
            <div>Edges: {edges.length}</div>
            {nestingMode && selectedParentId && (
              <div className="text-yellow-600 dark:text-yellow-400 font-medium">
                🎯 Nesting: {nodes.find(n => n.id === selectedParentId)?.data.label}
              </div>
            )}
          </div>
          
          {/* Scrollable palette area */}
          <div className="flex-1 overflow-y-auto">
            <LogiconPalette 
              onAddLogiconFlow={addLogiconFlow} 
              onAddLogiconRandom={addLogiconRandom}
            />
          </div>
        </div>

        {/* Canvas - Restore proper sizing */}
  <div className="flex-1 bg-gray-100 dark:bg-gray-900 relative overflow-hidden">
          {nestingMode && selectedParentId && (
            <div className="absolute top-4 right-4 z-10 bg-yellow-100 dark:bg-yellow-900 border border-yellow-300 dark:border-yellow-700 rounded-lg p-3 shadow-lg max-w-xs">
              <p className="text-sm text-yellow-800 dark:text-yellow-200 font-medium">
                🪆 Nesting Mode Active
              </p>
              <p className="text-xs text-yellow-600 dark:text-yellow-400 mb-2">
                Parent: {nodes.find(n => n.id === selectedParentId)?.data.label}
              </p>
              <p className="text-xs text-yellow-700 dark:text-yellow-300 mb-1">
                • Click logicons to add as nested children
              </p>
              <p className="text-xs text-yellow-700 dark:text-yellow-300 mb-1">
                • Click nodes to change parent selection
              </p>
              <p className="text-xs text-yellow-700 dark:text-yellow-300">
                • Drag edge endpoints to change connections
              </p>
            </div>
          )}
          
          <ReactFlow
            key={`reactflow-${currentDiagramName}-${nodes.length}`}
            nodes={nodes
              .filter((node, index, array) => {
                // Remove duplicate IDs - keep the first occurrence
                return array.findIndex((n: Node) => n.id === node.id) === index;
              })
              .map(node => {
              let className = '';
              let isInNestingGroup = false;
              let isNestingParent = false;
              let isCurrentNestingTarget = false;
              
              // Check if this node is part of a nesting relationship
              const isParent = nestingRelations.some(rel => rel.parentId === node.id);
              const isChild = nestingRelations.some(rel => rel.childId === node.id);
              isInNestingGroup = isParent || isChild;
              isNestingParent = isParent;
              isCurrentNestingTarget = nestingMode && selectedParentId === node.id;
              
              if (nestingMode && selectedParentId === node.id) {
                // In nesting mode, highlight the selected parent
                className = 'ring-4 ring-yellow-400 ring-opacity-60';
              } else if (!nestingMode && selectedNodeId) {
                // In normal mode, highlight the selected node and its group
                if (selectedNodeId === node.id) {
                  className = 'ring-4 ring-blue-400 ring-opacity-60';
                } else {
                  // Check if this node is a child of the selected parent
                  const children = getChildrenOf(selectedNodeId);
                  if (children.some(child => child.childId === node.id)) {
                    className = 'ring-2 ring-blue-300 ring-opacity-40';
                  }
                }
              }
              
              return {
                ...node,
                className,
                data: {
                  ...node.data,
                  isInNestingGroup,
                  isNestingParent,
                  isCurrentNestingTarget,
                },
              };
            })}
            edges={edges.filter((edge, index, array) => {
              // Remove duplicate edge IDs - keep the first occurrence
              return array.findIndex((e: Edge) => e.id === edge.id) === index;
            })}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onEdgeUpdate={onEdgeUpdate}
            onNodeClick={onNodeClick}
            onNodeContextMenu={onNodeContextMenu}
            nodeTypes={nodeTypes}
            defaultViewport={{ x: 0, y: 0, zoom: 1.0 }}
            fitView={false}
            className="bg-gray-100 dark:bg-gray-900"
          >
            <MiniMap 
              nodeColor={(node) => {
                switch (node.data?.category) {
                  case 'control': return '#3b82f6';      // Blue
                  case 'array': return '#10b981';        // Green  
                  case 'comparison': return '#8b5cf6';   // Purple
                  case 'math': return '#f59e0b';         // Orange
                  case 'string': return '#06b6d4';       // Cyan
                  case 'value': return '#10b981';        // Green
                  case 'file': return '#f59e0b';         // Orange
                  case 'date': return '#ef4444';         // Red
                  case 'crypto': return '#8b5cf6';       // Purple
                  case 'system': return '#6b7280';       // Gray
                  case 'couchbase': return '#ec4899';    // Pink
                  case 'dispatcher': return '#84cc16';   // Lime
                  case 'etl': return '#f97316';          // Orange
                  case 'host': return '#6366f1';         // Indigo
                  case 'json': return '#eab308';         // Yellow
                  case 'node': return '#22c55e';         // Green
                  case 'sql': return '#a855f7';          // Purple
                  case 'tree': return '#059669';         // Emerald
                  default: return '#6b7280';             // Gray
                }
              }}
              className="bg-white dark:bg-gray-800"
            />
            <Controls className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700" />
            <Background 
              variant={BackgroundVariant.Dots} 
              gap={12} 
              size={1} 
              color={theme === 'dark' ? '#374151' : '#e5e7eb'}
            />
          </ReactFlow>

          {/* Subflow overlays for group nodes */}
          {nodes.filter(n => n.type === 'group').map(g => {
            const sf = getSubflow(g.id);
            if (!sf) return null;
            const style: any = g.style || {};
            const bounds = {
              x: g.position.x,
              y: g.position.y,
              width: Number(style.width || 300),
              height: Number(style.height || 200),
            };
            return (
              <SubflowFrame key={`sf-${g.id}`} groupId={g.id} bounds={bounds} />
            );
          })}
          {contextMenu && (
            <ContextMenu
              x={contextMenu.x}
              y={contextMenu.y}
              onDelete={() => deleteNode(contextMenu.nodeId)}
              onClose={() => setContextMenu(null)}
              onProperties={() => openPropertiesDialog(contextMenu.nodeId)}
              nodeLabel={contextMenu.nodeLabel}
              isGroupRoot={contextMenu.isGroupRoot}
            />
          )}
          
          {/* Properties Dialog */}
          {propertiesDialog && propertiesDialog.nodeType === 'start' && (
            <StartNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as StartNodeProperties || { name: 'NameOfRule' }}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'declare' && (
            <DeclareNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as DeclareNodeProperties || { 
                isGlobal: false, 
                variableName: 'myVar', 
                typeSpecifier: 'S' 
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'if' && (
            <IfNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              initialProperties={propertiesDialog.properties as IfNodeProperties || {}}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'iif' && (
            <IifNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              initialProperties={propertiesDialog.properties as IifNodeProperties || {}}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'while' && (
            <WhileNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              initialProperties={propertiesDialog.properties as WhileNodeProperties || {}}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'cbQuery' && (
            <CBQueryNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              initialProperties={propertiesDialog.properties as CBQueryNodeProperties || {}}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'create' && (
            <CreateNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as CreateNodeProperties || { 
                nodeName: '' 
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'newTree' && (
            <NewTreeNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as NewTreeNodeProperties || {
                nodeName: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'csvNode' && (
            <CSVNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as CSVNodeProperties || {
                filename: '',
                delimiter: ',',
                hasHeaders: true
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'jsonNode' && (
            <JSONNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: JSONNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as JSONNodeProperties || {
                jsonOrName: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'xmlNode' && (
            <XMLNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: XMLNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as XMLNodeProperties || {
                xmlString: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'yamlNode' && (
            <YAMLNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: YAMLNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as YAMLNodeProperties || {
                yamlString: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'mapNode' && (
            <MapNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: MapNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as MapNodeProperties || {
                mapString: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'list' && (
            <ListNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: ListNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as ListNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'nodeToString' && (
            <NodeToStringNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: NodeToStringNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as NodeToStringNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'findByName' && (
            <FindByNameNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as FindByNameNodeProperties || {
                node: '',
                name: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'traverseNode' && (
            <TraverseNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: TraverseNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TraverseNodeProperties || {
                node: '',
                functionName: 'visitFn'
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'queryNode' && (
            <QueryNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: QueryNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as QueryNodeProperties || {
                node: '',
                functionName: 'predicateFn'
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'firstChild' && (
            <FirstChildNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as FirstChildNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'lastChild' && (
            <LastChildNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as LastChildNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'getAttribute' && (
            <GetAttributeNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as GetAttributeNodeProperties || {
                variableName: '',
                attributeName: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'setAttribute' && (
            <SetAttributeNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: SetAttributeProps) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as SetAttributeProps || {
                variableName: '',
                attributeName: '',
                value: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'removeAttribute' && (
            <RemoveAttributeNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: RemoveAttributeNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as RemoveAttributeNodeProperties || {
                variableName: '',
                attributeName: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'setAttributes' && (
            <SetAttributesNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: SetAttributesNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as SetAttributesNodeProperties || {
                variableName: '',
                attributesMap: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'getChildAt' && (
            <GetChildAtNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as GetChildAtNodeProperties || {
                node: '',
                index: 0
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'getChildByName' && (
            <GetChildByNameNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as GetChildByNameNodeProperties || {
                node: '',
                name: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'getDepth' && (
            <GetDepthNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: GetDepthNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as GetDepthNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'getLevel' && (
            <GetLevelNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: GetLevelNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as GetLevelNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'getName' && (
            <GetNameNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: GetNameNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as GetNameNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'setName' && (
            <SetNameNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: SetNameNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as SetNameNodeProperties || {
                variableName: '',
                name: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'getParent' && (
            <GetParentNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: GetParentNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as GetParentNodeProperties || {
                node: ''
              }}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'getPath' && (
            <GetPathNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: GetPathNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as GetPathNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'getRoot' && (
            <GetRootNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: GetRootNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as GetRootNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'getSiblings' && (
            <GetSiblingsNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: GetSiblingsNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as GetSiblingsNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'getText' && (
            <GetTextNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: GetTextNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as GetTextNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'setText' && (
            <SetTextNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: SetTextNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as SetTextNodeProperties || {
                variableName: '',
                text: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'hasAttribute' && (
            <HasAttributeNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: HasAttributeNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as HasAttributeNodeProperties || {
                variableName: '',
                attributeName: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'isLeaf' && (
            <IsLeafNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: IsLeafNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as IsLeafNodeProperties || {
                node: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'isRoot' && (
            <IsRootNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: IsRootNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as IsRootNodeProperties || {
                node: ''
              }}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'parseJSON' && (
            <ParseJSONNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as ParseJSONNodeProperties || { 
                jsonString: '{"key": "value"}',
                nodeName: 'root'
              }}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'array' && (
            <ArrayNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as ArrayNodeProperties || { 
                values: ['']
              }}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'addChild' && (
            <AddChildNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as AddChildNodeProperties || { 
                parentNode: '',
                childNode: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'removeChild' && (
            <RemoveChildNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties: RemoveChildNodeProperties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as RemoveChildNodeProperties || { 
                parentNode: '',
                childNode: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'childCount' && (
            <ChildCountNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as ChildCountNodeProperties || {
                node: 'node',
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'clear' && (
            <ClearNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as ClearNodeProperties || {
                node: 'node',
              }}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'treeSave' && (
            <TreeSaveNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TreeSaveNodeProperties || { 
                treeVariable: '',
                filename: '',
                format: 'json',
                compress: false
              }}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'treeLoad' && (
            <TreeLoadNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TreeLoadNodeProperties || { 
                filename: ''
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'treeSaveSecure' && (
            <TreeSaveSecureNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TreeSaveSecureNodeProperties || {
                treeVariable: 'tree',
                filename: 'secure.json',
                encryptionKeyID: 'encKey',
                signingKeyID: 'signKey',
                watermark: 'watermark',
                checksum: true,
                auditTrail: true,
                compressionLevel: 9,
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'treeLoadSecure' && (
            <TreeLoadSecureNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TreeLoadSecureNodeProperties || {
                filename: 'secure.json',
                decryptionKeyID: 'decKey',
                verificationKeyID: 'verifyKey',
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'treeValidateSecure' && (
            <TreeValidateSecureNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TreeValidateSecureNodeProperties || {
                filename: 'secure.json',
                verificationKeyID: 'verifyKey',
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'treeGetMetadata' && (
            <TreeGetMetadataNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TreeGetMetadataNodeProperties || {
                filename: 'data.json',
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'treeWalk' && (
            <TreeWalkNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TreeWalkNodeProperties || {
                treeVariable: 'tree',
                functionName: 'myFunc',
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'treeFind' && (
            <TreeFindNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TreeFindNodeProperties || { 
                treeVariable: '',
                fieldName: 'id',
                value: '',
                operator: '',
                searchAll: false
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'treeSearch' && (
            <TreeSearchNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TreeSearchNodeProperties || { 
                treeVariable: 'tree',
                fieldName: 'name',
                value: '',
                operator: '',
                existsOnly: false
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'treeToYAML' && (
            <TreeToYAMLNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TreeToYAMLNodeProperties || {
                treeVariable: 'tree',
              }}
            />
          )}

          {propertiesDialog && propertiesDialog.nodeType === 'treeToXML' && (
            <TreeToXMLNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as TreeToXMLNodeProperties || {
                treeVariable: 'tree',
                prettyPrint: true,
              }}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'addTo' && (
            <AddToNodePropertiesDialog
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              onDelete={() => {
                deleteNode(propertiesDialog.nodeId);
                setPropertiesDialog(null);
              }}
              initialProperties={propertiesDialog.properties as AddToNodeProperties || { 
                collectionName: '',
                value: ''
              }}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'logPrint' && (
            <LogPrintNodeProperties
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              initialProperties={propertiesDialog.properties as LogPrintProperties || { 
                message: '',
                logLevel: 'info',
                additionalArgs: []
              }}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'createTransform' && (
            <CreateTransformNodeProperties
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              initialProperties={propertiesDialog.properties as CreateTransformProperties || { 
                transformName: ''
              }}
            />
          )}
          
          {propertiesDialog && propertiesDialog.nodeType === 'addMapping' && (
            <AddMappingNodeProperties
              isOpen={true}
              onClose={() => setPropertiesDialog(null)}
              onSave={(properties) => saveNodeProperties(propertiesDialog.nodeId, properties)}
              initialProperties={propertiesDialog.properties as AddMappingProperties || { 
                transform: '',
                sourceField: '',
                targetColumn: '',
                program: [],
                dataType: 'string',
                required: false
              }}
            />
          )}
          
          {/* Fallback for unknown node types */}
          {propertiesDialog && propertiesDialog.nodeType === 'unknown' && (
            <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
              <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
                <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Properties Not Available
                  </h3>
                  <button
                    onClick={() => setPropertiesDialog(null)}
                    className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
                  >
                    ×
                  </button>
                </div>
                <div className="p-6">
                  <p className="text-sm text-gray-700 dark:text-gray-300 mb-4">
                    Properties dialog not yet implemented for this node type.
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">
                    Node ID: {propertiesDialog.nodeId}<br/>
                    Node Type: {propertiesDialog.nodeType}
                  </p>
                  <div className="flex gap-3">
                    <button
                      onClick={() => setPropertiesDialog(null)}
                      className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200 rounded"
                    >
                      Close
                    </button>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
        </div>
      </div>
    </ReactFlowProvider>
  );
}
