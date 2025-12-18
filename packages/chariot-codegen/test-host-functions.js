// Quick test for callMethod, getHostObject, hostObject code generation
import { generateChariotCodeFromDiagram } from './dist/index.js';

const diagram = {
  name: 'test-host',
  nodes: [
    {
      id: '1',
      type: 'logicon',
      data: {
        label: 'Call Method',
        icon: '📞',
        category: 'host',
        properties: { 
          objectName: 'myObj',
          methodName: 'DoSomething',
          args: "42, 'hello'"
        }
      },
      position: { x: 0, y: 0 }
    },
    {
      id: '2',
      type: 'logicon',
      data: {
        label: 'Get Host Object',
        icon: '🖥️',
        category: 'host',
        properties: { objectName: 'myObj' }
      },
      position: { x: 0, y: 100 }
    },
    {
      id: '3',
      type: 'logicon',
      data: {
        label: 'Host Object',
        icon: '🔗',
        category: 'host',
        properties: { 
          objectName: 'myObj',
          wrappedObject: 'someGoStruct'
        }
      },
      position: { x: 0, y: 200 }
    },
    {
      id: '4',
      type: 'logicon',
      data: {
        label: 'Host Object',
        icon: '🔗',
        category: 'host',
        properties: { 
          objectName: 'emptyObj',
          wrappedObject: ''
        }
      },
      position: { x: 0, y: 300 }
    }
  ],
  edges: [
    { id: 'e1-2', source: '1', target: '2' },
    { id: 'e2-3', source: '2', target: '3' },
    { id: 'e3-4', source: '3', target: '4' }
  ],
  nestingRelations: []
};

const code = generateChariotCodeFromDiagram(JSON.stringify(diagram), { embedSource: false });

console.log('Generated Code:');
console.log(code);
console.log('\n--- Verification ---');
console.log("✓ callMethod(myObj, 'DoSomething', 42, 'hello'):", 
  code.includes("callMethod(myObj, 'DoSomething', 42, 'hello')"));
console.log("✓ getHostObject('myObj'):", 
  code.includes("getHostObject('myObj')"));
console.log("✓ hostObject('myObj', someGoStruct):", 
  code.includes("hostObject('myObj', someGoStruct)"));
console.log("✓ hostObject('emptyObj') - no second arg:", 
  code.includes("hostObject('emptyObj')") && !code.includes("hostObject('emptyObj',"));
