import type { ReactNode } from 'react';

/**
 * SnapshotData is the payload carried in a state_snapshot or state_delta event.
 */
export interface SnapshotData {
  runId: string;
  taskId?: string;
  snapshotType: string;
  payload: Record<string, unknown>;
  delta?: Record<string, unknown>; // only for state_delta
}

/**
 * RenderContext provides shared rendering context to snapshot renderers.
 */
export interface RenderContext {
  runId: string;
  taskId?: string;
  messageId?: string;
  /** Whether this snapshot is rendered in the detail panel vs inline. */
  location: 'inline' | 'panel';
}

/**
 * A SnapshotRenderer renders a structured snapshot into a React node.
 */
export type SnapshotRenderer = (
  data: SnapshotData,
  ctx: RenderContext,
) => ReactNode;

const registry = new Map<string, SnapshotRenderer>();

/**
 * Register a renderer for a snapshot type. Overwrites any existing registration.
 */
export function registerSnapshotRenderer(
  type: string,
  renderer: SnapshotRenderer,
): void {
  registry.set(type, renderer);
}

/**
 * Get the renderer registered for the given snapshot type.
 */
export function getSnapshotRenderer(
  type: string,
): SnapshotRenderer | undefined {
  return registry.get(type);
}

/**
 * Check if a renderer is registered for the given type.
 */
export function hasSnapshotRenderer(type: string): boolean {
  return registry.has(type);
}

/**
 * Remove all registered renderers (useful for testing).
 */
export function clearSnapshotRegistry(): void {
  registry.clear();
}
