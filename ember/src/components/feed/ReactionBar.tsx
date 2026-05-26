import { useState } from "react";
import type { ReactionKind } from "@pulse/drift/types";
import { REACTION_LABELS } from "@pulse/drift/types";
import { reactToContent, removeReaction } from "../../api/content";
import { trackReaction } from "../../api/events";

const REACTION_CONFIG: Record<
  ReactionKind,
  { label: string; emoji: string; colorVar: string }
> = {
  gave_me_energy: {
    label: "Energy",
    emoji: "⚡",
    colorVar: "--color-accent-energy",
  },
  calmed_me: {
    label: "Calm",
    emoji: "🌊",
    colorVar: "--color-accent-calm",
  },
  on_repeat: {
    label: "Repeat",
    emoji: "🔁",
    colorVar: "--color-accent-repeat",
  },
  surprised_me: {
    label: "Surprise",
    emoji: "✨",
    colorVar: "--color-accent-surprise",
  },
  my_aesthetic: {
    label: "Aesthetic",
    emoji: "⭐",
    colorVar: "--color-accent-aesthetic",
  },
};

const REACTION_KINDS = Object.keys(REACTION_LABELS) as ReactionKind[];

interface Props {
  contentId: string;
  initialCounts?: Record<string, number>;
}

export default function ReactionBar({ contentId, initialCounts }: Props) {
  const [counts, setCounts] = useState<Record<string, number>>(
    initialCounts ?? {},
  );
  const [activeKinds, setActiveKinds] = useState<Set<ReactionKind>>(new Set());
  const [busyKind, setBusyKind] = useState<ReactionKind | null>(null);

  const toggleReaction = async (kind: ReactionKind) => {
    if (busyKind) return;

    setBusyKind(kind);
    const isActive = activeKinds.has(kind);

    try {
      if (isActive) {
        await removeReaction(contentId, kind);
        trackReaction(contentId, kind, false);
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
        trackReaction(contentId, kind, true);
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
    <div
      className="mt-3 flex flex-wrap gap-1.5"
      aria-label="Semantic reactions"
    >
      {REACTION_KINDS.map((kind) => {
        const isActive = activeKinds.has(kind);
        const count = counts[kind] ?? 0;
        const config = REACTION_CONFIG[kind];
        return (
          <button
            key={kind}
            type="button"
            title={REACTION_LABELS[kind]}
            onClick={() => toggleReaction(kind)}
            disabled={busyKind !== null}
            className={`text-xs px-2.5 py-1.5 rounded-full transition-all active:scale-95 ${
              isActive
                ? "text-white shadow-sm"
                : "bg-[var(--color-surface)] text-[var(--color-text-muted)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface-hover)]"
            }`}
            style={
              isActive
                ? {
                    backgroundColor: `var(${config.colorVar})`,
                  }
                : undefined
            }
          >
            {config.emoji}
            {count > 0 && <span className="ml-1 tabular-nums">{count}</span>}
          </button>
        );
      })}
    </div>
  );
}
