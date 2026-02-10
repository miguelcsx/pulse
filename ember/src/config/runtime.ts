function parsePositiveInt(raw: string | undefined, fallback: number): number {
  const parsed = Number.parseInt(String(raw ?? "").trim(), 10);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return fallback;
  }
  return parsed;
}

export const runtimeConfig = {
  mediaReadyTimeoutMs: parsePositiveInt(import.meta.env.VITE_MEDIA_READY_TIMEOUT_MS, 120000),
  mediaReadyPollIntervalMs: parsePositiveInt(import.meta.env.VITE_MEDIA_READY_POLL_MS, 1500),
} as const;

