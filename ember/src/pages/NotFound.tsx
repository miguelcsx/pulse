import { Link } from "react-router-dom";

export default function NotFound() {
  return (
    <div className="min-h-dvh flex items-center justify-center bg-[var(--color-bg)] p-6">
      <div className="max-w-sm w-full text-center space-y-6">
        <div className="space-y-2">
          <p className="text-5xl font-semibold text-[var(--color-text-muted)]">
            404
          </p>
          <h1 className="text-xl font-semibold text-[var(--color-text)]">
            Page not found
          </h1>
          <p className="text-sm text-[var(--color-text-muted)]">
            The page you&rsquo;re looking for doesn&rsquo;t exist.
          </p>
        </div>
        <Link
          to="/"
          className="inline-block px-6 py-2.5 rounded-[var(--radius-sm)] bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-[var(--color-bg)] text-sm font-medium transition-colors"
        >
          Go home
        </Link>
      </div>
    </div>
  );
}
