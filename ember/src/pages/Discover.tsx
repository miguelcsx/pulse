import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import type { TodayResponse } from "@pulse/drift/types";
import { getToday } from "../api/advice";
import BridgeCard from "../components/advice/BridgeCard";
import Spinner from "../components/ui/Spinner";
import { usePageTitle } from "../hooks/usePageTitle";
import { useUiStore } from "../store/uiStore";

export default function Discover() {
  usePageTitle("Discover");
  const [data, setData] = useState<TodayResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const addToast = useUiStore((s) => s.addToast);

  useEffect(() => {
    let cancelled = false;
    getToday()
      .then((res) => {
        if (!cancelled) setData(res);
      })
      .catch(() => addToast("Failed to load", "error"))
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [addToast]);

  if (loading) {
    return (
      <div className="flex justify-center py-16">
        <Spinner size="lg" />
      </div>
    );
  }

  const bridges = data?.bridges ?? [];
  return (
    <div className="space-y-8 pb-4">
      {/* Header */}
      <section className="pt-4">
        <h1 className="text-[28px] font-semibold tracking-tight">Discover</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Your network built from asks and help, not followers.
        </p>
      </section>

      {/* Stats */}
      <section className="grid grid-cols-2 gap-3">
        {[
          { value: bridges.length, label: "Bridges" },
          { value: data?.trust_profile?.helped_count ?? 0, label: "Helped" },
        ].map((stat) => (
          <div
            key={stat.label}
            className="rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border)] p-4 text-center"
          >
            <p className="text-2xl font-semibold tabular-nums">{stat.value}</p>
            <p className="text-xs text-[var(--color-text-muted)] mt-0.5">
              {stat.label}
            </p>
          </div>
        ))}
      </section>

      {/* Bridges */}
      <section className="space-y-3">
        <h2 className="text-[17px] font-semibold">Bridges</h2>
        {bridges.length === 0 ? (
          <div className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border)] p-6 text-center">
            <p className="text-sm text-[var(--color-text-muted)] mb-3">
              No bridges yet. Start by asking a question.
            </p>
            <Link
              to="/"
              className="inline-flex rounded-[var(--radius-sm)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white hover:bg-[var(--color-accent-hover)] transition-colors"
            >
              Ask something
            </Link>
          </div>
        ) : (
          bridges.map((bridge) => (
            <BridgeCard key={bridge.id} bridge={bridge} />
          ))
        )}
      </section>
    </div>
  );
}
