export function getWebSocketURL(): string {
  const configuredBase = import.meta.env.VITE_WS_BASE?.trim();
  if (configuredBase) {
    const base = configuredBase.replace(/\/+$/, "");
    return `${base}/ws`;
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/ws`;
}
