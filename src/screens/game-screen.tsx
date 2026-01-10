import React from "react";
import { Box, Text } from "ink";
import type { GameState } from "../types.js";

interface GameScreenProps {
  state: GameState;
}

/**
 * Main game screen with map view, resource bar, and prompt window.
 * TODO: Implement full game UI (Phase 1+)
 */
export function GameScreen({ state }: GameScreenProps): React.ReactElement {
  return (
    <Box flexDirection="column" padding={1}>
      {/* Resource Bar */}
      <Box borderStyle="single" paddingX={1} marginBottom={1}>
        <Box marginRight={2}>
          <Text>
            Model: <Text color="green">{state.model === "opus" ? "Opus" : "Sonnet"}</Text>
          </Text>
        </Box>
        <Box marginRight={2}>
          <Text>
            Context: <Text color="yellow">{state.contextTokens.toLocaleString()}</Text>
            <Text dimColor>/{state.maxContextTokens.toLocaleString()}</Text>
          </Text>
        </Box>
        <Box>
          <Text>
            Cost: <Text color="cyan">${state.apiCost.toFixed(4)}</Text>
          </Text>
        </Box>
      </Box>

      {/* Main Area (Map placeholder) */}
      <Box flexDirection="row" height={15}>
        {/* Advisor Panel (left) */}
        <Box
          width={20}
          borderStyle="single"
          flexDirection="column"
          paddingX={1}
        >
          <Text bold>Advisors</Text>
          <Text dimColor>No advisors configured</Text>
        </Box>

        {/* Map View (center) */}
        <Box
          flexGrow={1}
          borderStyle="single"
          flexDirection="column"
          alignItems="center"
          justifyContent="center"
        >
          <Text bold>Map View</Text>
          <Text dimColor>Project: {state.project.name}</Text>
          <Text dimColor>{state.project.path}</Text>
          <Box marginTop={1}>
            <Text color="gray">[ Map visualization coming in Phase 1 ]</Text>
          </Box>
        </Box>

        {/* Mission Panel (right) */}
        <Box
          width={25}
          borderStyle="single"
          flexDirection="column"
          paddingX={1}
        >
          <Text bold>Missions</Text>
          <Text dimColor>No active missions</Text>
        </Box>
      </Box>

      {/* Prompt Window (bottom) */}
      <Box
        borderStyle="single"
        paddingX={1}
        marginTop={1}
        flexDirection="column"
      >
        <Text bold>Prompt</Text>
        <Text dimColor>Press Enter to focus, type your request, Enter to submit</Text>
        <Box marginTop={1}>
          <Text color="cyan">&gt; </Text>
          <Text>_</Text>
        </Box>
      </Box>

      {/* Status Bar */}
      <Box marginTop={1}>
        <Text dimColor>
          [h/j/k/l] Navigate | [1-5] Views | [Enter] Prompt | [a] Advisors | [?] Help
        </Text>
      </Box>
    </Box>
  );
}
