## 1. **Chariot Runtime Support**

You’ll need to enhance the Chariot runtime to:
- Track the current executing line or statement (in the script or parsed AST).
- Maintain a mapping from source lines to AST nodes and runtime state.
- Support breakpoints, step-in/step-over, and continue operations.
- Expose the current variable scope and call stack.

**Implementation Steps:**
- **Parser/AST:** Annotate each AST node with its source line/column.
- **Interpreter:** At each step, update a `currentLine`/`currentNode` property in the runtime.
- **Breakpoints:** Add a mechanism to pause execution at specified lines or function entries.
- **Variable Inspection:** Expose the current scope (variables and their values) at any pause point.

---

## 2. **Chariot Server Debug APIs**

Expose debugging endpoints over HTTP/WebSocket, such as:
- `POST /debug/breakpoints` — Set/clear breakpoints.
- `POST /debug/step` — Step in/over/out.
- `POST /debug/continue` — Resume execution.
- `GET /debug/state` — Get current line, call stack, and variable values.
- `GET /debug/source` — Get the source code for display.

The server should be able to pause and resume script execution on demand, and send events (e.g., "paused at line X") to the IDE.

---

## 3. **Charioteer IDE Integration**

The IDE can:
- Display the source code with line numbers.
- Highlight the current execution line (based on `/debug/state`).
- Show a live variable panel (from `/debug/state`).
- Allow the user to set breakpoints, step, and continue (via debug APIs).
- Optionally, display the call stack and function arguments.

---

## 4. **Feasibility and Effort**

- **Parser/AST changes:** Straightforward; just annotate nodes with line/column info.
- **Interpreter changes:** Moderate; need to check for breakpoints and update state at each step.
- **Server APIs:** Moderate; mostly wrappers around runtime state and control.
- **IDE integration:** Standard for modern editors; can use WebSocket for live updates.

**Similar systems:**  
This is how debuggers for Python, JavaScript, and even many Lisp interpreters work. Chariot’s interpreted, AST-driven design makes this easier than for compiled languages.

---

## 5. **Summary Table**

| Feature                | Chariot Runtime | Chariot Server | Charioteer IDE |
|------------------------|-----------------|---------------|---------------|
| Source line tracking   | Yes             | Yes           | Yes           |
| Breakpoints            | Yes             | Yes           | Yes           |
| Step/Continue          | Yes             | Yes           | Yes           |
| Variable inspection    | Yes             | Yes           | Yes           |
| Call stack             | Yes (optional)  | Yes           | Yes           |
| Source display         | N/A             | Yes           | Yes           |

---

## 6. **Next Steps**

- Annotate AST nodes with source positions in `parser.go` and `ast.go`.
- Add a debug state object to the runtime.
- Implement pause/resume/step logic in the interpreter.
- Add debug endpoints to the server.
- Update Charioteer to use these APIs for UI.

---

**Conclusion:**  
Chariot is well-suited for source-level debugging. With moderate effort, you can provide a modern debugging experience in Charioteer, including source highlighting and live variable inspection.

