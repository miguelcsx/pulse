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
      className="flex items-center justify-center gap-2 bg-amber-900/60 border-b border-amber-700/50 px-4 py-1.5 text-xs text-amber-200"
    >
      <span className="relative flex h-2 w-2 shrink-0">
        <span className="absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75 animate-ping" />
        <span className="relative inline-flex h-2 w-2 rounded-full bg-amber-500" />
      </span>
      <span>Reconnecting&hellip;</span>
    </div>
  );
}
