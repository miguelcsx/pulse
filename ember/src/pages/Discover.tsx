import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import type { TodayResponse } from "@pulse/drift/types";
import { getToday } from "../api/advice";
import BridgeCard from "../components/advice/BridgeCard";
import Spinner from "../components/ui/Spinner";
import { usePageTitle } from "../hooks/usePageTitle";
import { useUiStore } from "../store/uiStore";

export default function Discover() {
  usePageTitle("Affinity Map");
  const [data, setData] = useState<TodayResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const addToast = useUiStore((s) => s.addToast);

  useEffect(() => {
    let cancelled = false;
    getToday()
      .then((res) => {
        if (!cancelled) setData(res);
      })
      .catch(() => addToast("Failed to load affinity map", "error"))
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [addToast]);

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    );
  }

  const bridges = data?.bridges ?? [];
  const sessions = data?.help_sessions ?? [];

  return (
    <div className="space-y-7">
      <section>
        <p className="text-xs font-semibold uppercase tracking-[0.12em] text-[var(--color-text-muted)]">
          Affinity map
        </p>
        <h1 className="mt-2 text-2xl font-semibold">
          Your current advice graph
        </h1>
        <p className="mt-2 text-sm text-[var(--color-text-muted)]">
          Bridges, live contexts, and your trust profile are built from asks and
          help signals, not follower-count ranking.
        </p>
      </section>

      <section className="grid gap-3 sm:grid-cols-3">
        <div className="rounded-lg border border-[var(--color-border)] p-4">
          <p className="text-2xl font-semibold">{bridges.length}</p>
          <p className="text-xs text-[var(--color-text-muted)]">active bridges</p>
        </div>
        <div className="rounded-lg border border-[var(--color-border)] p-4">
          <p className="text-2xl font-semibold">{sessions.length}</p>
          <p className="text-xs text-[var(--color-text-muted)]">live rooms</p>
        </div>
        <div className="rounded-lg border border-[var(--color-border)] p-4">
          <p className="text-2xl font-semibold">
            {data?.trust_profile?.helped_count ?? 0}
          </p>
          <p className="text-xs text-[var(--color-text-muted)]">people helped</p>
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Human bridges</h2>
        {bridges.length === 0 ? (
          <div className="rounded-lg border border-dashed border-[var(--color-border)] p-5">
            <p className="text-sm text-[var(--color-text-muted)]">
              No bridges yet. Start with an ask on Today.
            </p>
            <Link
              to="/"
              className="mt-3 inline-flex rounded-lg bg-[var(--color-primary)] px-4 py-2 text-sm font-medium text-white"
            >
              Create ask
            </Link>
          </div>
        ) : (
          bridges.map((bridge) => <BridgeCard key={bridge.id} bridge={bridge} />)
        )}
      </section>

      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Live contexts</h2>
        <div className="grid gap-3 sm:grid-cols-3">
          {sessions.map((session) => (
            <article
              key={session.id}
              className="rounded-lg border border-[var(--color-border)] p-4"
            >
              <p className="text-sm font-semibold">{session.title}</p>
              <p className="mt-2 text-xs leading-relaxed text-[var(--color-text-muted)]">
                {session.description}
              </p>
              <p className="mt-3 text-xs text-[var(--color-text-muted)]">
                {session.member_count} inside
              </p>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
