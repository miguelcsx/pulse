import { useEffect } from "react";
import { Link } from "react-router-dom";
import { useFeedContextStore } from "../../store/feedContextStore";

export default function ContextSheet() {
  const activeRoom = useFeedContextStore((s) => s.activeRoom);
  const open = useFeedContextStore((s) => s.contextSheetOpen);
  const closeSheet = useFeedContextStore((s) => s.closeSheet);

  useEffect(() => {
    if (!open) return;
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") closeSheet();
    };
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [open, closeSheet]);

  if (!open || !activeRoom) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 backdrop-blur-sm"
      onClick={(e) => {
        if (e.target === e.currentTarget) closeSheet();
      }}
    >
      <div className="w-full max-w-xl rounded-t-[var(--radius-xl)] bg-[var(--color-bg-elevated)] border-t border-x border-[var(--color-border)] p-6 pb-8 animate-slide-up">
        <div className="mx-auto mb-4 h-1 w-10 rounded-full bg-[var(--color-surface-active)]" />

        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">Shared context</h3>
          <button
            onClick={closeSheet}
            className="rounded-full p-1.5 text-[var(--color-text-muted)] hover:bg-[var(--color-surface)] hover:text-[var(--color-text)] transition-colors"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-4 w-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div className="flex flex-wrap gap-2 mb-4">
          {activeRoom.tags.map((tag) => (
            <span
              key={tag}
              className="px-3 py-1 rounded-full bg-[var(--color-surface)] text-[var(--color-text-secondary)] text-sm"
            >
              #{tag}
            </span>
          ))}
        </div>

        <p className="text-sm text-[var(--color-text-muted)] mb-6">
          Pulse groups nearby moments by shared tags. This is a related feed,
          not a chat room.
        </p>

        <Link
          to={`/rooms/${activeRoom.room_id}`}
          onClick={closeSheet}
          className="block w-full text-center px-4 py-3 rounded-[var(--radius-sm)] bg-[var(--color-accent)] text-white font-medium text-sm hover:bg-[var(--color-accent-hover)] transition-colors"
        >
          View related moments
        </Link>
      </div>
    </div>
  );
}
