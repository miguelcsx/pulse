export default function Footer() {
  return (
    <footer className="border-t border-[var(--color-border)] py-8 px-6">
      <div className="max-w-5xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4">
        <span className="text-sm text-[var(--color-text-muted)]">
          <span className="bg-gradient-to-r from-[var(--color-primary)] to-pink-500 bg-clip-text text-transparent font-semibold">
            Pulse
          </span>
          {" "}&mdash; the human layer after AI
        </span>
        <span className="text-xs text-[var(--color-text-muted)]">
          &copy; {new Date().getFullYear()} Pulse. Built for human advice.
        </span>
      </div>
    </footer>
  );
}
