import { Link } from "react-router-dom";

export default function NotFound() {
  return (
    <div className="min-h-dvh flex items-center justify-center bg-[var(--color-bg)] p-6">
      <div className="max-w-sm w-full text-center space-y-6">
        <div className="space-y-2">
          <p className="text-6xl font-bold text-indigo-400">404</p>
          <h1 className="text-xl font-semibold text-[var(--color-text)]">
            Page not found
          </h1>
          <p className="text-sm text-[var(--color-text-muted)]">
            The page you&rsquo;re looking for doesn&rsquo;t exist or has been
            moved.
          </p>
        </div>
        <Link
          to="/"
          className="inline-block px-6 py-2.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium transition-colors"
        >
          Back to Feed
        </Link>
      </div>
    </div>
  );
}
