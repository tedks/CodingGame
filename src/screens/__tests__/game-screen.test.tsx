import { describe, it, expect } from "vitest";
import React from "react";
import { render } from "ink-testing-library";
import { GameScreen } from "../game-screen.js";
import type { GameState } from "../../types.js";

describe("GameScreen", () => {
  const mockGameState: GameState = {
    model: "sonnet",
    project: {
      name: "test-project",
      path: "/path/to/test-project",
    },
    contextTokens: 5000,
    maxContextTokens: 200000,
    apiCost: 0.1234,
  };

  it("renders resource bar with model information", () => {
    const { lastFrame } = render(<GameScreen state={mockGameState} />);

    expect(lastFrame()).toContain("Model:");
    expect(lastFrame()).toContain("Sonnet");
  });

  it("displays context token usage correctly", () => {
    const { lastFrame } = render(<GameScreen state={mockGameState} />);

    expect(lastFrame()).toContain("Context:");
    expect(lastFrame()).toContain("5,000");
    expect(lastFrame()).toContain("200,000");
  });

  it("shows API cost formatted to 4 decimal places", () => {
    const { lastFrame } = render(<GameScreen state={mockGameState} />);

    expect(lastFrame()).toContain("Cost:");
    expect(lastFrame()).toContain("$0.1234");
  });

  it("renders all main UI sections", () => {
    const { lastFrame } = render(<GameScreen state={mockGameState} />);

    expect(lastFrame()).toContain("Advisors");
    expect(lastFrame()).toContain("Map View");
    expect(lastFrame()).toContain("Missions");
    expect(lastFrame()).toContain("Prompt");
  });

  it("displays project information", () => {
    const { lastFrame } = render(<GameScreen state={mockGameState} />);

    expect(lastFrame()).toContain("test-project");
    expect(lastFrame()).toContain("/path/to/test-project");
  });

  it("shows keyboard shortcuts in status bar", () => {
    const { lastFrame } = render(<GameScreen state={mockGameState} />);

    expect(lastFrame()).toContain("[h/j/k/l] Navigate");
    expect(lastFrame()).toContain("[Enter] Prompt");
  });

  it("displays Opus model correctly", () => {
    const opusState: GameState = {
      ...mockGameState,
      model: "opus",
    };
    const { lastFrame } = render(<GameScreen state={opusState} />);

    expect(lastFrame()).toContain("Opus");
  });
});
