import { appHref } from "../config";

export default function Hero() {
  return (
    <section className="pt-32 pb-20 px-6">
      <div className="max-w-3xl mx-auto text-center">
        <h1 className="text-5xl md:text-6xl font-bold leading-tight">
          Connect Through
          <span className="text-indigo-400"> Moments</span>
        </h1>
        <p className="mt-6 text-lg text-[var(--color-text-muted)] max-w-xl mx-auto">
          Pulse is a social platform that helps you discover people with shared taste.
          No vanity metrics, no engagement traps, just affinity-driven connection.
        </p>
        <div className="mt-10 flex gap-4 justify-center">
          <a
            href={appHref("/register")}
            className="px-8 py-3 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium transition-colors"
          >
            Start Exploring
          </a>
          <a
            href="#features"
            className="px-8 py-3 rounded-lg border border-[var(--color-border)] hover:border-indigo-500 text-[var(--color-text)] transition-colors"
          >
            Learn More
          </a>
        </div>
      </div>
    </section>
  );
}
