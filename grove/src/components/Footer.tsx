export default function Footer() {
  return (
    <footer className="border-t border-[var(--color-border)] py-8 px-6">
      <div className="max-w-5xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4">
        <span className="text-sm text-[var(--color-text-muted)]">
          Pulse &mdash; Connect through moments
        </span>
        <span className="text-xs text-[var(--color-text-muted)]">
          Built for social discovery, by builders
        </span>
      </div>
    </footer>
  );
}
