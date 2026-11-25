import React from "react";
import VisualDSLPrototype from "./pages/Canvas";
import { ThemeProvider } from "./contexts/ThemeContext";
import { FlowControlProvider } from "./contexts/FlowControlContext";
import { NestingProvider } from "./contexts/NestingContext";
import { readDiagramDraft } from "./utils/diagramDraft";

function App() {
  // Read once before Canvas AND its contexts mount. No empty first render can
  // race autosave, and restoring is not confused with an explicit file import.
  const [initialDraft] = React.useState(readDiagramDraft);
  return (
    <ThemeProvider>
      <FlowControlProvider initialDraft={initialDraft}>
        <NestingProvider initialDraft={initialDraft}>
          <VisualDSLPrototype initialDraft={initialDraft} />
        </NestingProvider>
      </FlowControlProvider>
    </ThemeProvider>
  );
}

export default App;

