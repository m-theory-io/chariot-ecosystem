// Quick test for sleep, getEnv, exit code generation
import { generateChariotCodeFromDiagram } from './dist/index.js';

const diagram = {
  name: 'test',
  nodes: [
    {
      id: '1',
      type: 'logicon',
      data: {
        label: 'Sleep',
        icon: '😴',
        category: 'system',
        properties: { milliseconds: '100' }
      },
      position: { x: 0, y: 0 }
    },
    {
      id: '2',
      type: 'logicon',
      data: {
        label: 'Get Env',
        icon: '🌐',
        category: 'system',
        properties: { varName: 'PATH' }
      },
      position: { x: 0, y: 100 }
    },
    {
      id: '3',
      type: 'logicon',
      data: {
        label: 'Exit',
        icon: '🚪',
        category: 'system',
        properties: { exitCode: '0' }
      },
      position: { x: 0, y: 200 }
    }
  ],
  edges: [
    { id: 'e1-2', source: '1', target: '2' },
    { id: 'e2-3', source: '2', target: '3' }
  ],
  nestingRelations: []
};

const code = generateChariotCodeFromDiagram(JSON.stringify(diagram), { embedSource: false });

console.log('Generated Code:');
console.log(code);
console.log('\n--- Verification ---');
console.log('✓ sleep(100) - numeric, unquoted:', code.includes('sleep(100)'));
console.log('✓ getEnv(\'PATH\') - string quoted:', code.includes("getEnv('PATH')"));
console.log('✓ exit() - no argument:', code.includes('exit()'));
console.log('✗ sleep(\'100\') - should NOT be quoted:', code.includes("sleep('100')"));
console.log('✗ getenv - should NOT be lowercase:', code.includes('getenv'));
