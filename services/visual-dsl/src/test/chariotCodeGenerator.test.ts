import { generateChariotCodeFromDiagram } from 'chariot-codegen';

// Test the code generator with the usersAgent diagram
const testUsersAgentGeneration = () => {
  // Sample diagram data (simplified version of usersAgent.json)
  const sampleDiagram = {
    "name": "usersAgent",
    "nodes": [
      {
        "id": "start",
        "type": "logicon",
        "position": { "x": 50, "y": 50 },
        "data": {
          "label": "Start",
          "icon": "🚀",
          "category": "control"
        }
      },
      {
        "id": "declare-1",
        "type": "logicon",
        "position": { "x": 50, "y": 200 },
        "data": {
          "label": "Declare",
          "icon": "📋",
          "category": "value",
          "properties": {
            "isGlobal": true,
            "variableName": "usersAgent",
            "typeSpecifier": "T"
          }
        }
      },
      {
        "id": "create-2",
        "type": "logicon",
        "position": { "x": 50, "y": 350 },
        "data": {
          "label": "Create",
          "icon": "🆕",
          "category": "node",
          "properties": {
            "nodeName": "usersAgent"
          }
        }
      },
      {
        "id": "declare-3",
        "type": "logicon",
        "position": { "x": 276, "y": 211 },
        "data": {
          "label": "Declare",
          "icon": "📋",
          "category": "value",
          "properties": {
            "isGlobal": false,
            "variableName": "users",
            "typeSpecifier": "J"
          }
        }
      },
      {
        "id": "parseJSON-4",
        "type": "logicon",
        "position": { "x": 276, "y": 361 },
        "data": {
          "label": "Parse JSON",
          "icon": "📖",
          "category": "json",
          "properties": {
            "jsonString": "[]",
            "nodeName": "users"
          }
        }
      },
      {
        "id": "addChild-12",
        "type": "logicon",
        "position": { "x": 1100, "y": 244 },
        "data": {
          "label": "Add Child",
          "icon": "➕",
          "category": "node"
        }
      },
      {
        "id": "treeSave-16",
        "type": "logicon",
        "position": { "x": 1757, "y": 248 },
        "data": {
          "label": "Tree Save",
          "icon": "💾",
          "category": "tree",
          "properties": {
            "filename": "usersAgent.json"
          }
        }
      }
    ],
    "edges": [
      {
        "id": "start-declare-1",
        "source": "start",
        "target": "declare-1"
      },
      {
        "id": "declare-1-create-2",
        "source": "declare-1",
        "target": "create-2"
      },
      {
        "id": "declare-1-declare-3",
        "source": "declare-1",
        "target": "declare-3"
      },
      {
        "id": "declare-3-parseJSON-4",
        "source": "declare-3",
        "target": "parseJSON-4"
      },
      {
        "id": "declare-3-addChild-12",
        "source": "declare-3",
        "target": "addChild-12"
      },
      {
        "id": "addChild-12-treeSave-16",
        "source": "addChild-12",
        "target": "treeSave-16"
      }
    ],
    "nestingRelations": [
      {
        "parentId": "declare-1",
        "childId": "create-2",
        "order": 0
      },
      {
        "parentId": "declare-3",
        "childId": "parseJSON-4",
        "order": 0
      }
    ]
  };

  try {
    const diagramJson = JSON.stringify(sampleDiagram);
    const generatedCode = generateChariotCodeFromDiagram(diagramJson);
    
    console.log('Generated Chariot Code:');
    console.log('='.repeat(50));
    console.log(generatedCode);
    console.log('='.repeat(50));
    
    return generatedCode;
  } catch (error) {
    console.error('Code generation failed:', error);
    return null;
  }
};

// Run the test in browser environment
if (typeof window !== 'undefined') {
  console.log('Testing Chariot Code Generator...');
  testUsersAgentGeneration();
}

export { testUsersAgentGeneration };
