import React from "react";
import VisualDSLPrototype from "./pages/Canvas";
import { ThemeProvider } from "./contexts/ThemeContext";
import { FlowControlProvider } from "./contexts/FlowControlContext";
import { NestingProvider } from "./contexts/NestingContext";

function App() {
  return (
    <ThemeProvider>
      <FlowControlProvider>
        <NestingProvider>
          <VisualDSLPrototype />
        </NestingProvider>
      </FlowControlProvider>
    </ThemeProvider>
  );
}

export default App;

