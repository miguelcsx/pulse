import { appHref } from "../config";

export default function Hero() {
  return (
    <section className="relative px-6 pb-16 pt-28 overflow-hidden">
      <div className="relative mx-auto grid max-w-6xl gap-10 lg:grid-cols-[1fr_28rem] lg:items-center">
        <div>
          <p className="text-sm font-semibold uppercase tracking-[0.16em] text-[var(--color-text-muted)]">
            Human advice network
          </p>
        <h1
          className="mt-4 text-5xl md:text-6xl font-bold leading-tight animate-fade-up"
        >
          AI gives answers. Pulse finds the human who lived it.
        </h1>
        <p
          className="mt-6 text-lg text-[var(--color-text-muted)] max-w-xl animate-fade-up"
          style={{ animationDelay: "0.15s" }}
        >
          Ask for perspective, get explainable bridges to mentors and peers,
          then talk while the context is still alive.
        </p>
        <div
          className="mt-10 flex flex-wrap gap-4 animate-fade-up"
          style={{ animationDelay: "0.3s" }}
        >
          <a
            href={appHref("/register")}
            className="px-8 py-3 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white font-medium transition-all hover:shadow-[0_0_20px_rgba(99,102,241,0.4)]"
          >
            Find a human
          </a>
          <a
            href="#features"
            className="px-8 py-3 rounded-lg border border-[var(--color-border)] hover:border-[var(--color-border-emphasis)] text-[var(--color-text)] transition-colors"
          >
            See how it works
          </a>
        </div>
        </div>

        <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-2xl animate-fade-up">
          <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-bg)] p-4">
            <p className="text-xs font-semibold uppercase tracking-[0.12em] text-[var(--color-text-muted)]">
              Today
            </p>
            <div className="mt-3 rounded-lg border border-[var(--color-border)] p-3">
              <p className="text-sm text-[var(--color-text-muted)]">
                What do you need human perspective on?
              </p>
              <p className="mt-3 text-sm">
                I am launching my first product and need advice getting the
                first 10 real users.
              </p>
            </div>
            <div className="mt-4 space-y-3">
              {[
                ["Mentor", "Built a marketplace and struggled with cold start."],
                ["Peer", "Also testing founder-led sales this week."],
                ["Adjacent", "Ran community launches for creative tools."],
              ].map(([label, reason]) => (
                <div
                  key={label}
                  className="rounded-lg border border-[var(--color-border)] p-3"
                >
                  <div className="flex items-center justify-between gap-3">
                    <span className="text-sm font-semibold">{label}</span>
                    <span className="rounded-full bg-[var(--color-primary)] px-2 py-0.5 text-xs text-white">
                      Ask
                    </span>
                  </div>
                  <p className="mt-2 text-xs text-[var(--color-text-muted)]">
                    {reason}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
