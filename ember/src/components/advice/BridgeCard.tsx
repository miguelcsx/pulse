import { useState } from "react";
import { Link } from "react-router-dom";
import type { Bridge, BridgeStatus, HelpSignalKind } from "@pulse/drift/types";
import Button from "../ui/Button";
import { askBridge, signalBridge } from "../../api/advice";
import { useUiStore } from "../../store/uiStore";

const bridgeLabels = {
  mentor: "Mentor",
  peer: "Peer",
  adjacent_perspective: "Adjacent",
} as const;

const signalLabels: Array<{ kind: HelpSignalKind; label: string }> = [
  { kind: "useful", label: "Useful" },
  { kind: "practical", label: "Practical" },
  { kind: "clarifying", label: "Clarifying" },
  { kind: "not_relevant", label: "Skip" },
];

interface Props {
  bridge: Bridge;
  onUpdate?: (bridge: Bridge) => void;
}

export default function BridgeCard({ bridge, onUpdate }: Props) {
  const [status, setStatus] = useState<BridgeStatus>(bridge.status);
  const [busy, setBusy] = useState(false);
  const addToast = useUiStore((s) => s.addToast);

  async function handleAsk() {
    setBusy(true);
    try {
      const updated = await askBridge(bridge.id);
      setStatus(updated.status);
      onUpdate?.(updated);
      addToast("Bridge opened", "success");
    } catch {
      addToast("Failed to open bridge", "error");
    } finally {
      setBusy(false);
    }
  }

  async function handleSignal(kind: HelpSignalKind) {
    try {
      await signalBridge(bridge.id, kind);
      if (kind === "not_relevant") {
        setStatus("dismissed");
      }
      addToast("Signal recorded", "success");
    } catch {
      addToast("Failed to record signal", "error");
    }
  }

  if (status === "dismissed") {
    return null;
  }

  const user = bridge.recommended_user;

  return (
    <article className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4">
      <div className="flex items-start gap-3">
        <Link
          to={`/profile/${user.id}`}
          className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--color-surface-active)] text-sm font-semibold text-[var(--color-text-secondary)]"
        >
          {user.display_name?.[0] || user.handle?.[0] || "?"}
        </Link>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <Link
              to={`/profile/${user.id}`}
              className="truncate text-sm font-semibold hover:text-[var(--color-accent)] transition-colors"
            >
              {user.display_name || user.handle}
            </Link>
            <span className="rounded-full bg-[var(--color-surface)] px-2 py-0.5 text-[11px] font-medium text-[var(--color-text-muted)]">
              {bridgeLabels[bridge.bridge_type]}
            </span>
            <span className="text-[11px] tabular-nums text-[var(--color-text-muted)]">
              {Math.round(bridge.confidence * 100)}%
            </span>
          </div>
          <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
            @{user.handle}
          </p>
          <p className="mt-2 text-sm leading-relaxed text-[var(--color-text-secondary)]">
            {bridge.reason}
          </p>
        </div>
      </div>

      <div className="mt-4 flex flex-wrap items-center gap-2">
        <Button size="sm" variant="accent" onClick={handleAsk} loading={busy}>
          {status === "asked" ? "Asked" : "Ask"}
        </Button>
        <Link
          to={`/profile/${user.id}`}
          className="inline-flex items-center justify-center rounded-[var(--radius-sm)] bg-[var(--color-surface)] px-3 py-1.5 text-[13px] font-medium hover:bg-[var(--color-surface-hover)] transition-colors"
        >
          View profile
        </Link>
        <div className="ml-auto flex gap-1">
          {signalLabels.map((signal) => (
            <button
              key={signal.kind}
              type="button"
              onClick={() => handleSignal(signal.kind)}
              className="rounded-full px-2 py-1 text-[11px] text-[var(--color-text-muted)] hover:bg-[var(--color-surface)] hover:text-[var(--color-text)] transition-colors"
            >
              {signal.label}
            </button>
          ))}
        </div>
      </div>
    </article>
  );
}
