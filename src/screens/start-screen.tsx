import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import SelectInput from "ink-select-input";
import type { Model, Project } from "../types.js";

interface StartScreenProps {
  onStart: (model: Model, project: Project) => void;
}

type Step = "model" | "project";

const MODEL_OPTIONS = [
  { label: "Claude 4.5 Opus - More capable, higher cost", value: "opus" as Model },
  { label: "Claude 4.5 Sonnet - Faster, lower cost", value: "sonnet" as Model },
];

/**
 * Game start screen for configuration.
 * Allows selecting model and project before entering the game.
 */
export function StartScreen({ onStart }: StartScreenProps): React.ReactElement {
  const [step, setStep] = useState<Step>("model");
  const [selectedModel, setSelectedModel] = useState<Model | null>(null);

  useInput((input, key) => {
    if (key.escape) {
      if (step === "project") {
        setStep("model");
        setSelectedModel(null);
      }
    }
  });

  const handleModelSelect = (item: { value: Model }): void => {
    setSelectedModel(item.value);
    setStep("project");
  };

  const handleProjectSelect = (item: { value: string }): void => {
    if (selectedModel) {
      onStart(selectedModel, {
        path: item.value,
        name: item.value.split("/").pop() || "project",
      });
    }
  };

  // TODO: Replace with actual recent projects from config
  const PROJECT_OPTIONS = [
    { label: "Current directory (.)", value: process.cwd() },
    { label: "Open other project...", value: "__browse__" },
  ];

  return (
    <Box flexDirection="column" padding={1}>
      <Box marginBottom={1}>
        <Text bold color="cyan">
          ╔═══════════════════════════════════════╗
        </Text>
      </Box>
      <Box marginBottom={1}>
        <Text bold color="cyan">
          ║         C O D I N G G A M E           ║
        </Text>
      </Box>
      <Box marginBottom={1}>
        <Text bold color="cyan">
          ╚═══════════════════════════════════════╝
        </Text>
      </Box>

      <Box marginBottom={1}>
        <Text dimColor>
          Strategy game interface for Claude Code
        </Text>
      </Box>

      <Box marginTop={1} flexDirection="column">
        {step === "model" && (
          <>
            <Box marginBottom={1}>
              <Text bold>Select Model:</Text>
            </Box>
            <SelectInput items={MODEL_OPTIONS} onSelect={handleModelSelect} />
          </>
        )}

        {step === "project" && (
          <>
            <Box marginBottom={1}>
              <Text>
                Model: <Text color="green">{selectedModel === "opus" ? "Opus" : "Sonnet"}</Text>
              </Text>
            </Box>
            <Box marginBottom={1}>
              <Text bold>Select Project:</Text>
            </Box>
            <SelectInput items={PROJECT_OPTIONS} onSelect={handleProjectSelect} />
            <Box marginTop={1}>
              <Text dimColor>Press ESC to go back</Text>
            </Box>
          </>
        )}
      </Box>

      <Box marginTop={2}>
        <Text dimColor>
          Use arrow keys to navigate, Enter to select
        </Text>
      </Box>
    </Box>
  );
}
