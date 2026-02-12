import { useWsStore } from "../../store/wsStore";
import { useAuthStore } from "../../store/authStore";

export default function ConnectionStatus() {
  const connected = useWsStore((s) => s.connected);
  const isAuthenticated = useAuthStore((s) => !!s.accessToken);

  // Only show disconnect banner when the user is logged in but WS is down
  if (!isAuthenticated || connected) return null;

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
