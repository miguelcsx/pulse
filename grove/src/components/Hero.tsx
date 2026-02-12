import { appHref } from "../config";

export default function Hero() {
  return (
    <section className="relative pt-32 pb-20 px-6 overflow-hidden">
      {/* Animated gradient blobs */}
      <div
        className="absolute top-20 -left-32 w-72 h-72 bg-indigo-500/30 rounded-full blur-[128px] animate-blob-pulse"
        aria-hidden="true"
      />
      <div
        className="absolute top-40 -right-32 w-72 h-72 bg-pink-500/20 rounded-full blur-[128px] animate-blob-pulse"
        style={{ animationDelay: "2s" }}
        aria-hidden="true"
      />

      <div className="relative max-w-3xl mx-auto text-center">
        <h1
          className="text-5xl md:text-6xl font-bold leading-tight animate-fade-up"
        >
          Connect Through
          <span className="bg-gradient-to-r from-[var(--color-primary)] to-pink-500 bg-clip-text text-transparent">
            {" "}Moments
          </span>
        </h1>
        <p
          className="mt-6 text-lg text-[var(--color-text-muted)] max-w-xl mx-auto animate-fade-up"
          style={{ animationDelay: "0.15s" }}
        >
          Pulse is a social platform that helps you discover people with shared taste.
          No vanity metrics, no engagement traps, just affinity-driven connection.
        </p>
        <div
          className="mt-10 flex gap-4 justify-center animate-fade-up"
          style={{ animationDelay: "0.3s" }}
        >
          <a
            href={appHref("/register")}
            className="px-8 py-3 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white font-medium transition-all hover:shadow-[0_0_20px_rgba(99,102,241,0.4)]"
          >
            Start Exploring
          </a>
          <a
            href="#features"
            className="px-8 py-3 rounded-lg border border-[var(--color-border)] hover:border-[var(--color-border-emphasis)] text-[var(--color-text)] transition-colors"
          >
            Learn More
          </a>
        </div>
      </div>
    </section>
  );
}
