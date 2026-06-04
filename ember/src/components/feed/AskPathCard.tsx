import { useState } from "react";
import type { Bridge } from "@pulse/drift/types";
import Button from "../ui/Button";
import { respondBridge } from "../../api/advice";
import { useUiStore } from "../../store/uiStore";

interface Props {
  bridge: Bridge;
  onUpdate?: (bridge: Bridge) => void;
}

export default function AskPathCard({ bridge, onUpdate }: Props) {
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const addToast = useUiStore((s) => s.addToast);
  const asker = bridge.ask?.user;
  const responded = bridge.status === "responded";
  const response = bridge.responses?.[0];

  async function handleRespond() {
    const message = draft.trim();
    if (message.length < 8) {
      addToast("Add a short perspective first", "error");
      return;
    }
    setBusy(true);
    try {
      const updated = await respondBridge(bridge.id, message);
      onUpdate?.(updated);
      setDraft("");
      addToast("Perspective offered", "success");
    } catch {
      addToast("Failed to offer perspective", "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <article className="snap-start bg-[var(--color-bg-elevated)] rounded-none sm:rounded-[var(--radius-lg)] overflow-hidden border-y sm:border border-[var(--color-border)]">
      <div className="p-4">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--color-surface-active)] text-sm font-semibold text-[var(--color-text-secondary)]">
            {asker?.display_name?.[0] || asker?.handle?.[0] || "?"}
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-text-muted)]">
              Affinity path
            </p>
            <h2 className="mt-1 text-[17px] font-semibold leading-snug">
              Someone could use your perspective
            </h2>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              {asker?.display_name || asker?.handle || "Someone"} appears here
              because their ask overlaps with your context.
            </p>
          </div>
        </div>

        <div className="mt-4 rounded-[var(--radius-md)] bg-[var(--color-surface)] p-4">
          <p className="text-sm leading-relaxed">{bridge.ask?.question}</p>
        </div>

        <p className="mt-3 text-sm leading-relaxed text-[var(--color-text-secondary)]">
          {bridge.reason}
        </p>

        {response ? (
          <div className="mt-4 rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg)] p-3">
            <p className="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-text-muted)]">
              Your perspective
            </p>
            <p className="mt-2 text-sm leading-relaxed whitespace-pre-wrap">
              {response.body}
            </p>
          </div>
        ) : (
          <div className="mt-4 space-y-3">
            <textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="Share the one thing you would tell them..."
              maxLength={1200}
              className="min-h-24 w-full resize-none rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg)] p-3 text-sm leading-relaxed outline-none placeholder:text-[var(--color-text-muted)] focus:border-[var(--color-accent)]"
            />
            <div className="flex justify-end">
              <Button
                size="sm"
                variant={responded ? "secondary" : "accent"}
                disabled={responded}
                loading={busy}
                onClick={handleRespond}
              >
                {responded ? "Perspective offered" : "Offer perspective"}
              </Button>
            </div>
          </div>
        )}
      </div>
    </article>
  );
}
