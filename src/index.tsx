#!/usr/bin/env node
/**
 * CodingGame - Strategy game UI wrapper for Claude Code
 *
 * A Civilization/Factorio-inspired interface for software development.
 * This is a game interface to coding, not gamification.
 */

import React from "react";
import { render } from "ink";
import { App } from "./app.js";

// Entry point
render(<App />);
