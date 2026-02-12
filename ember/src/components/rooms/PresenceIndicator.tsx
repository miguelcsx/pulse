interface Props {
  count: number;
}

export default function PresenceIndicator({ count }: Props) {
  return (
    <div className="flex items-center gap-2">
      <span className="relative flex h-3 w-3">
        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-[var(--color-success)] opacity-75"></span>
        <span className="relative inline-flex rounded-full h-3 w-3 bg-[var(--color-success)]"></span>
      </span>
      <span className="text-sm text-[var(--color-text-muted)]">
        {count} {count === 1 ? "person" : "people"} in this room
      </span>
    </div>
  );
}
