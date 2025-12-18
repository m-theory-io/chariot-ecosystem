// Quick test for apply, clone, contains code generation
import { generateChariotCodeFromDiagram } from './dist/index.js';

const diagram = {
  name: 'test-dispatcher',
  nodes: [
    {
      id: '1',
      type: 'logicon',
      data: {
        label: 'Apply',
        icon: '🎯',
        category: 'dispatcher',
        properties: { 
          functionName: 'myFunc',
          collection: 'myArray'
        }
      },
      position: { x: 0, y: 0 }
    },
    {
      id: '2',
      type: 'logicon',
      data: {
        label: 'Clone',
        icon: '👥',
        category: 'dispatcher',
        properties: { 
          object: 'myTree',
          newName: 'copyName'
        }
      },
      position: { x: 0, y: 100 }
    },
    {
      id: '3',
      type: 'logicon',
      data: {
        label: 'Clone',
        icon: '👥',
        category: 'dispatcher',
        properties: { 
          object: 'myArray',
          newName: ''
        }
      },
      position: { x: 0, y: 200 }
    },
    {
      id: '4',
      type: 'logicon',
      data: {
        label: 'Contains',
        icon: '🔍',
        category: 'dispatcher',
        properties: { 
          object: 'myString',
          value: 'll'
        }
      },
      position: { x: 0, y: 300 }
    },
    {
      id: '5',
      type: 'logicon',
      data: {
        label: 'Get All Meta',
        icon: '📊',
        category: 'dispatcher',
        properties: { 
          target: 'myNode'
        }
      },
      position: { x: 0, y: 400 }
    },
    {
      id: '6',
      type: 'logicon',
      data: {
        label: 'Get At',
        icon: '📍',
        category: 'dispatcher',
        properties: { 
          target: 'myArray',
          index: '2'
        }
      },
      position: { x: 0, y: 500 }
    },
    {
      id: '7',
      type: 'logicon',
      data: {
        label: 'Get Attributes',
        icon: '🏷️',
        category: 'dispatcher',
        properties: { 
          target: 'myNode'
        }
      },
      position: { x: 0, y: 600 }
    },
    {
      id: '8',
      type: 'logicon',
      data: {
        label: 'Get Meta',
        icon: '📋',
        category: 'dispatcher',
        properties: { 
          target: 'myNode',
          key: 'createdBy'
        }
      },
      position: { x: 0, y: 700 }
    },
    {
      id: '9',
      type: 'logicon',
      data: {
        label: 'Get Property',
        icon: '🔑',
        category: 'dispatcher',
        properties: { 
          target: 'myMap',
          key: 'name'
        }
      },
      position: { x: 0, y: 800 }
    },
    {
      id: '10',
      type: 'logicon',
      data: {
        label: 'Index Of',
        icon: '🔢',
        category: 'dispatcher',
        properties: { 
          target: 'myString',
          value: 'na',
          startIndex: '2'
        }
      },
      position: { x: 0, y: 900 }
    },
    {
      id: '11',
      type: 'logicon',
      data: {
        label: 'Set Meta',
        icon: '📝',
        category: 'dispatcher',
        properties: {
          target: 'myNode',
          key: 'lastUpdated',
          value: 'approvedFlag'
        }
      },
      position: { x: 0, y: 1000 }
    },
    {
      id: '12',
      type: 'logicon',
      data: {
        label: 'Set Property',
        icon: '🛠️',
        category: 'dispatcher',
        properties: {
          target: 'myMap',
          key: 'score',
          value: '42'
        }
      },
      position: { x: 0, y: 1100 }
    },
    {
      id: '13',
      type: 'logicon',
      data: {
        label: 'Add Mapping Transform',
        icon: '🔄',
        category: 'etl',
        properties: {
          transform: 'myTransform',
          sourceField: 'price',
          targetColumn: 'unit_price',
          transformName: 'moneyToDecimal',
          dataType: 'float',
          required: true,
          defaultValue: '0.0'
        }
      },
      position: { x: 0, y: 1200 }
    },
    {
      id: '14',
      type: 'logicon',
      data: {
        label: 'Do ETL',
        icon: '⚙️',
        category: 'etl',
        properties: {
          jobId: 'jobIdVar',
          csvFile: "'data/orders.csv'",
          transformConfig: 'myTransform',
          targetConfig: 'sqlTargetConfig',
          options: "map('batchSize', 500)"
        }
      },
      position: { x: 0, y: 1300 }
    },
    {
      id: '15',
      type: 'logicon',
      data: {
        label: 'ETL Status',
        icon: '📊',
        category: 'etl',
        properties: {
          jobId: 'jobIdVar'
        }
      },
      position: { x: 0, y: 1400 }
    }
  ],
  edges: [
    { id: 'e1-2', source: '1', target: '2' },
    { id: 'e2-3', source: '2', target: '3' },
    { id: 'e3-4', source: '3', target: '4' },
    { id: 'e4-5', source: '4', target: '5' },
    { id: 'e5-6', source: '5', target: '6' },
    { id: 'e6-7', source: '6', target: '7' },
    { id: 'e7-8', source: '7', target: '8' },
    { id: 'e8-9', source: '8', target: '9' },
    { id: 'e9-10', source: '9', target: '10' },
    { id: 'e10-11', source: '10', target: '11' },
    { id: 'e11-12', source: '11', target: '12' },
    { id: 'e12-13', source: '12', target: '13' },
    { id: 'e13-14', source: '13', target: '14' },
    { id: 'e14-15', source: '14', target: '15' }
  ],
  nestingRelations: []
};

const code = generateChariotCodeFromDiagram(JSON.stringify(diagram), { embedSource: false });

console.log('Generated Code:');
console.log(code);
console.log('\n--- Verification ---');
console.log("✓ apply(myFunc, myArray):", 
  code.includes("apply(myFunc, myArray)"));
console.log("✓ clone(myTree, 'copyName'):", 
  code.includes("clone(myTree, 'copyName')"));
console.log("✓ clone(myArray) - no newName:", 
  code.includes("clone(myArray)") && !code.includes("clone(myArray,"));
console.log("✓ contains(myString, 'll'):", 
  code.includes("contains(myString, 'll')"));
console.log("✓ getAllMeta(myNode):", 
  code.includes("getAllMeta(myNode)"));
console.log("✓ getAt(myArray, 2):", 
  code.includes("getAt(myArray, 2)"));
console.log("✓ getAttributes(myNode):", 
  code.includes("getAttributes(myNode)"));
console.log("✓ getMeta(myNode, 'createdBy'):", 
  code.includes("getMeta(myNode, 'createdBy')"));
console.log("✓ getProp(myMap, 'name'):", 
  code.includes("getProp(myMap, 'name')"));
console.log("✓ indexOf(myString, 'na', 2):", 
  code.includes("indexOf(myString, 'na', 2)"));
console.log("✓ setMeta(myNode, 'lastUpdated', approvedFlag):",
  code.includes("setMeta(myNode, 'lastUpdated', approvedFlag)"));
console.log("✓ setProp(myMap, 'score', 42):",
  code.includes("setProp(myMap, 'score', 42)"));
console.log("✓ addMappingWithTransform with default value:",
  code.includes("addMappingWithTransform(myTransform, 'price', 'unit_price', 'moneyToDecimal', 'float', true, '0.0')"));
console.log("✓ doETL(jobIdVar, 'data/orders.csv', myTransform, sqlTargetConfig, map('batchSize', 500)):",
  code.includes("doETL(jobIdVar, 'data/orders.csv', myTransform, sqlTargetConfig, map('batchSize', 500))"));
console.log("✓ etlStatus(jobIdVar):",
  code.includes("etlStatus(jobIdVar)"));
