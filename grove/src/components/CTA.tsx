import { appHref } from "../config";

export default function CTA() {
  return (
    <section className="py-20 px-6">
      <div className="max-w-2xl mx-auto text-center bg-[var(--color-surface)] rounded-2xl p-12 border border-[var(--color-border)]">
        <h2 className="text-3xl font-bold">Ready to find your people?</h2>
        <p className="mt-4 text-[var(--color-text-muted)]">
          Join a social platform where connections are built on shared creative signals, not vanity metrics.
        </p>
        <a
          href={appHref("/register")}
          className="inline-block mt-8 px-8 py-3 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium transition-colors"
        >
          Create Your Account
        </a>
      </div>
    </section>
  );
}
