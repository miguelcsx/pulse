import { useEffect, useRef, useState } from "react";
import { appHref } from "../config";

export default function CTA() {
  const sectionRef = useRef<HTMLElement>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const el = sectionRef.current;
    if (!el) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setVisible(true);
          observer.disconnect();
        }
      },
      { threshold: 0.3 },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return (
    <section ref={sectionRef} className="py-20 px-6">
      <div
        className={`max-w-2xl mx-auto rounded-2xl p-[1px] bg-gradient-to-r from-[var(--color-primary)] to-pink-500 ${
          visible ? "animate-fade-in" : "opacity-0"
        }`}
      >
        <div className="rounded-2xl bg-[var(--color-surface)] p-12 text-center">
          <h2 className="text-3xl font-bold">Ready to find your people?</h2>
          <p className="mt-4 text-[var(--color-text-muted)]">
            Join a social platform where connections are built on shared creative signals, not vanity metrics.
          </p>
          <a
            href={appHref("/register")}
            className="inline-block mt-8 px-8 py-3 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white font-medium transition-all hover:shadow-[0_0_20px_rgba(99,102,241,0.4)]"
          >
            Create Your Account
          </a>
        </div>
      </div>
    </section>
  );
}
