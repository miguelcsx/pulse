import { useState } from "react";
import type { ReactionKind } from "@pulse/drift/types";
import { REACTION_LABELS } from "@pulse/drift/types";
import { reactToContent, removeReaction } from "../../api/content";

const REACTION_SHORT_LABELS: Record<ReactionKind, string> = {
  gave_me_energy: "Energy",
  calmed_me: "Calm",
  on_repeat: "Repeat",
  surprised_me: "Surprised",
  my_aesthetic: "Aesthetic",
};

const REACTION_KINDS = Object.keys(REACTION_LABELS) as ReactionKind[];

interface Props {
  contentId: string;
  initialCounts?: Record<string, number>;
}

export default function ReactionBar({ contentId, initialCounts }: Props) {
  const [counts, setCounts] = useState<Record<string, number>>(initialCounts ?? {});
  const [activeKinds, setActiveKinds] = useState<Set<ReactionKind>>(new Set());
  const [busyKind, setBusyKind] = useState<ReactionKind | null>(null);

  const toggleReaction = async (kind: ReactionKind) => {
    if (busyKind) return;

    setBusyKind(kind);
    const isActive = activeKinds.has(kind);

    try {
      if (isActive) {
        await removeReaction(contentId, kind);
        setActiveKinds((prev) => {
          const next = new Set(prev);
          next.delete(kind);
          return next;
        });
        setCounts((prev) => ({
          ...prev,
          [kind]: Math.max((prev[kind] ?? 0) - 1, 0),
        }));
      } else {
        await reactToContent(contentId, kind);
        setActiveKinds((prev) => {
          const next = new Set(prev);
          next.add(kind);
          return next;
        });
        setCounts((prev) => ({
          ...prev,
          [kind]: (prev[kind] ?? 0) + 1,
        }));
      }
    } finally {
      setBusyKind(null);
    }
  };

  return (
    <div className="mt-3 flex flex-wrap gap-1.5" aria-label="Semantic reactions">
      {REACTION_KINDS.map((kind) => {
        const isActive = activeKinds.has(kind);
        const count = counts[kind] ?? 0;
        return (
          <button
            key={kind}
            type="button"
            title={REACTION_LABELS[kind]}
            onClick={() => toggleReaction(kind)}
            disabled={busyKind !== null}
            className={`text-xs px-2 py-1 rounded-full border transition-colors ${
              isActive
                ? "bg-indigo-600 border-indigo-600 text-white"
                : "border-[var(--color-border)] text-[var(--color-text-muted)] hover:border-indigo-500"
            }`}
          >
            {REACTION_SHORT_LABELS[kind]}
            {count > 0 && <span className="ml-1">{count}</span>}
          </button>
        );
      })}
    </div>
  );
}
