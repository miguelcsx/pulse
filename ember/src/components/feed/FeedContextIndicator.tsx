import { useFeedContextStore } from "../../store/feedContextStore";

export default function FeedContextIndicator() {
  const activeRoom = useFeedContextStore((s) => s.activeRoom);
  const openSheet = useFeedContextStore((s) => s.openSheet);

  if (!activeRoom) return null;

  const tagLabel =
    activeRoom.tags.length > 2
      ? `#${activeRoom.tags.slice(0, 2).join(" #")} +${activeRoom.tags.length - 2}`
      : activeRoom.tags.map((t) => `#${t}`).join(" ");

  return (
    <button
      type="button"
      onClick={openSheet}
      className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-[var(--color-surface)] border border-[var(--color-border)] text-xs font-medium transition-all animate-in"
    >
      <span className="relative flex h-2 w-2">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
        <span className="relative inline-flex h-2 w-2 rounded-full bg-emerald-500" />
      </span>
      <span className="truncate max-w-[120px]">{tagLabel}</span>
      <span className="text-[var(--color-text-muted)]">
        {activeRoom.member_count}
      </span>
    </button>
  );
}
