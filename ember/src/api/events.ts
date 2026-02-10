import type { AppEvent } from "@pulse/drift/types";
import client from "./client";

const FLUSH_INTERVAL_MS = 10_000;
const MAX_BATCH_SIZE = 50;
const MAX_QUEUE_SIZE = 500;

const queue: AppEvent[] = [];
let flushTimer: ReturnType<typeof setInterval> | null = null;
let flushing = false;
let started = false;

function now(): string {
  return new Date().toISOString();
}

export function trackEvent(
  type: string,
  targetType: string,
  targetId?: string,
  metadata?: Record<string, unknown>,
): void {
  const event: AppEvent = {
    type,
    target_type: targetType,
    target_id: targetId,
    metadata,
    timestamp: now(),
  };

  if (queue.length >= MAX_QUEUE_SIZE) {
    queue.shift();
  }

  queue.push(event);

  if (queue.length >= MAX_BATCH_SIZE) {
    void flushEvents();
  }
}

export async function flushEvents(): Promise<void> {
  if (flushing || queue.length === 0) return;

  flushing = true;
  const batch = queue.splice(0, MAX_BATCH_SIZE);

  try {
    await client.post("/events", batch);
  } catch {
    // Re-queue failed events at the front, but respect the cap
    const room = MAX_QUEUE_SIZE - queue.length;
    if (room > 0) {
      queue.unshift(...batch.slice(0, room));
    }
  } finally {
    flushing = false;
  }
}

export function startEventLoop(): void {
  if (started) return;
  started = true;

  flushTimer = setInterval(() => {
    void flushEvents();
  }, FLUSH_INTERVAL_MS);

  if (typeof window !== "undefined") {
    window.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "hidden") {
        void flushEvents();
      }
    });

    window.addEventListener("pagehide", () => {
      if (queue.length === 0) return;
      const batch = queue.splice(0, MAX_BATCH_SIZE);
      const blob = new Blob([JSON.stringify(batch)], {
        type: "application/json",
      });
      const url = client.defaults.baseURL ?? "/api/v1";
      navigator.sendBeacon(`${url}/events`, blob);
    });
  }
}

export function stopEventLoop(): void {
  if (flushTimer !== null) {
    clearInterval(flushTimer);
    flushTimer = null;
  }
  started = false;
  void flushEvents();
}

// ── Convenience helpers for common event types ──

export function trackView(targetType: string, targetId: string): void {
  trackEvent("view", targetType, targetId);
}

export function trackDwell(
  targetType: string,
  targetId: string,
  durationMs: number,
): void {
  trackEvent("dwell", targetType, targetId, { duration_ms: durationMs });
}

export function trackSkip(targetType: string, targetId: string): void {
  trackEvent("skip", targetType, targetId);
}

export function trackReaction(
  targetId: string,
  kind: string,
  added: boolean,
): void {
  trackEvent("reaction", "content", targetId, { kind, added });
}

export function trackSave(targetType: string, targetId: string): void {
  trackEvent("save", targetType, targetId);
}

export function trackPathFollow(pathId: string): void {
  trackEvent("path_follow", "path", pathId);
}

export function trackTagExplore(tagName: string): void {
  trackEvent("tag_explore", "tag", undefined, { tag: tagName });
}

export function trackSearch(query: string): void {
  trackEvent("search", "app", undefined, { query });
}
