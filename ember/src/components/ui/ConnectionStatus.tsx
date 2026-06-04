import { useEffect, useState } from "react";
import { useWsStore } from "../../store/wsStore";
import { useAuthStore } from "../../store/authStore";

// Only surface a disconnect after it has persisted this long, so brief
// reconnects (dev HMR, network blips, tab focus changes) never flash a banner.
const DISCONNECT_GRACE_MS = 4000;

export default function ConnectionStatus() {
  const connected = useWsStore((s) => s.connected);
  const isAuthenticated = useAuthStore((s) => !!s.accessToken);
  const [showBanner, setShowBanner] = useState(false);

  useEffect(() => {
    if (!isAuthenticated || connected) {
      setShowBanner(false);
      return;
    }
    const timer = setTimeout(() => setShowBanner(true), DISCONNECT_GRACE_MS);
    return () => clearTimeout(timer);
  }, [connected, isAuthenticated]);

  if (!showBanner) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      className="flex items-center justify-center gap-2 bg-[var(--color-surface)] border-b border-[var(--color-warning)] px-4 py-1.5 text-xs text-[var(--color-warning)]"
    >
      <span className="relative flex h-2 w-2 shrink-0">
        <span className="absolute inline-flex h-full w-full rounded-full bg-[var(--color-warning)] opacity-75 animate-ping" />
        <span className="relative inline-flex h-2 w-2 rounded-full bg-[var(--color-warning)]" />
      </span>
      <span>Reconnecting&hellip;</span>
    </div>
  );
}
