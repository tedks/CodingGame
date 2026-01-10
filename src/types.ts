/**
 * Core types for CodingGame
 */

/** Available Claude models */
export type Model = "opus" | "sonnet";

/** Project configuration */
export interface Project {
  /** Absolute path to project root */
  path: string;
  /** Project name (from package.json, Cargo.toml, etc.) */
  name: string;
}

/** Current game state */
export interface GameState {
  /** Selected Claude model */
  model: Model;
  /** Current project */
  project: Project;
  /** Current context token usage */
  contextTokens: number;
  /** Maximum context tokens for the model */
  maxContextTokens: number;
  /** Accumulated API cost in USD */
  apiCost: number;
}

/** Input mode for keyboard navigation */
export type InputMode = "normal" | "insert" | "visual";

/** Tile types for the map */
export type TileType = "file" | "directory" | "fogged";

/** Map tile representing a file or directory */
export interface Tile {
  /** File/directory path */
  path: string;
  /** Tile type */
  type: TileType;
  /** Whether this tile has been analyzed by Claude */
  revealed: boolean;
  /** Child tiles (for directories) */
  children?: Tile[];
}

/** Zoom levels for the map view */
export type ZoomLevel = 1 | 2 | 3 | 4 | 5;

export const ZOOM_LEVEL_NAMES: Record<ZoomLevel, string> = {
  1: "World",
  2: "Region",
  3: "City",
  4: "Street",
  5: "Interior",
};
