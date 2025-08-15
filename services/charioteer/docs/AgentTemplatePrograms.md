# Agent Template Programs
An agent template program defines the structural skeleton of an agent, the *agent prototype*. Running the program instantiates the prototype and saves it in the tree store. Once instantiated, other Chariot programs can interact with the agent:

- Access agent data nodes
- Call agent rule functions

The details of these actions are dependent on the implementation.  

Agent data that is read-only should be declared and initialized as part of the constructed agent prototype. Agent attributes that often vary at runtime, or that depend on external data sources, should be initialized, but not populated in the prototype.

### Use Cases
- Decision agent that only receives the ID of the profile to be evaluated. The ID must be used to query the configured database before applying the rules
- User account agent that does not save account data. Rules that need to access account must query the database.
- LLM agent that has configuration for interacting with external LLMs
- Supervisor agent that orchestrates collaboration among a complex of agents, either to accomplish an imperative job, or to seek a prompted goal, or a combination thereof

## Template Program Management and Reuse
Chariot programs are stored in two ways by the system.

1. As .ch files in the Charioteer user's local file system.
2. As "pretty printed" function definitions in the configured `sdtlib.json` in the Chariot Server

Once loaded into the editor, both .ch files and library functions may be executed. The runtime results of equivalent representations are the same, regardless of storage method.  Thus, the recommended workflow is:

- Create a new local .ch file to implement your program
- Test the execution of your file.
- When the new .ch file is ready, switch to Function Library to save the logic to the library

By saving reusable programs in the Function Library, a set of agent templates and helper functions can be loaded on Chariot Server startup.  
