import { appHref } from "../config";

export default function Navbar() {
  return (
    <nav className="fixed top-0 w-full bg-[var(--color-bg)]/80 backdrop-blur-md border-b border-[var(--color-border)] z-50">
      <div className="max-w-5xl mx-auto px-6 h-16 flex items-center justify-between">
        <span className="text-xl font-bold text-indigo-400">Pulse</span>
        <div className="flex items-center gap-4">
          <a
            href={appHref("/login")}
            className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
          >
            Log in
          </a>
          <a
            href={appHref("/register")}
            className="text-sm px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white transition-colors"
          >
            Get Started
          </a>
        </div>
      </div>
    </nav>
  );
}
