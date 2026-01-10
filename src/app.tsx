import React, { useState } from "react";
import { Box, Text } from "ink";
import { StartScreen } from "./screens/start-screen.js";
import { GameScreen } from "./screens/game-screen.js";
import type { GameState, Model, Project } from "./types.js";

/**
 * Main application component.
 * Manages overall game state and screen transitions.
 */
export function App(): React.ReactElement {
  const [screen, setScreen] = useState<"start" | "game">("start");
  const [gameState, setGameState] = useState<GameState | null>(null);

  const handleStartGame = (model: Model, project: Project): void => {
    setGameState({
      model,
      project,
      contextTokens: 0,
      maxContextTokens: model === "opus" ? 200000 : 200000,
      apiCost: 0,
    });
    setScreen("game");
  };

  if (screen === "start") {
    return <StartScreen onStart={handleStartGame} />;
  }

  if (!gameState) {
    return (
      <Box>
        <Text color="red">Error: No game state</Text>
      </Box>
    );
  }

  return <GameScreen state={gameState} />;
}
