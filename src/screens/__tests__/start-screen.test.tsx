import { describe, it, expect, vi } from "vitest";
import React from "react";
import { render } from "ink-testing-library";
import { StartScreen } from "../start-screen.js";
import type { Model, Project } from "../../types.js";

describe("StartScreen", () => {
  it("renders the title and initial model selection", () => {
    const onStart = vi.fn();
    const { lastFrame } = render(<StartScreen onStart={onStart} />);

    expect(lastFrame()).toContain("C O D I N G G A M E");
    expect(lastFrame()).toContain("Select Model:");
  });

  it("displays model options", () => {
    const onStart = vi.fn();
    const { lastFrame } = render(<StartScreen onStart={onStart} />);

    expect(lastFrame()).toContain("Claude 4.5 Opus");
    expect(lastFrame()).toContain("Claude 4.5 Sonnet");
  });

  it("shows project selection after model is selected", () => {
    const onStart = vi.fn();
    const { lastFrame, stdin } = render(<StartScreen onStart={onStart} />);

    // Simulate pressing Enter to select first model
    stdin.write("\r");

    expect(lastFrame()).toContain("Model:");
    expect(lastFrame()).toContain("Select Project:");
  });

  it("calls onStart with correct parameters when project is selected", () => {
    const onStart = vi.fn<[Model, Project], void>();
    const { stdin } = render(<StartScreen onStart={onStart} />);

    // Select model (first option - Opus)
    stdin.write("\r");

    // Select project (first option - current directory)
    stdin.write("\r");

    expect(onStart).toHaveBeenCalledWith(
      "opus",
      expect.objectContaining({
        path: expect.any(String),
        name: expect.any(String),
      })
    );
  });

  it("shows ESC hint on project selection screen", () => {
    const onStart = vi.fn();
    const { lastFrame, stdin } = render(<StartScreen onStart={onStart} />);

    // Navigate to project selection
    stdin.write("\r");

    expect(lastFrame()).toContain("Press ESC to go back");
  });
});
