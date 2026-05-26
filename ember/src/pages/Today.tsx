import { useEffect, useState } from "react";
import type {
  Bridge,
  DesiredHelpType,
  HelpSession,
  TodayResponse,
} from "@pulse/drift/types";
import { createAsk, getToday, joinHelpSession } from "../api/advice";
import BridgeCard from "../components/advice/BridgeCard";
import Button from "../components/ui/Button";
import Spinner from "../components/ui/Spinner";
import { usePageTitle } from "../hooks/usePageTitle";
import { useUiStore } from "../store/uiStore";

const quickIntents: Array<{ type: DesiredHelpType; label: string; helper: string }> = [
  { type: "advice", label: "Need advice", helper: "Find someone with lived context." },
  { type: "peer", label: "Find a peer", helper: "Compare notes with someone close." },
  { type: "mentor", label: "Find mentor", helper: "Talk to someone a few steps ahead." },
  { type: "feedback", label: "Get feedback", helper: "Invite an adjacent perspective." },
];

export default function Today() {
  usePageTitle("Today");
  const [data, setData] = useState<TodayResponse | null>(null);
  const [question, setQuestion] = useState("");
  const [helpType, setHelpType] = useState<DesiredHelpType>("advice");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const addToast = useUiStore((s) => s.addToast);

  useEffect(() => {
    let cancelled = false;
    getToday()
      .then((res) => {
        if (!cancelled) setData(res);
      })
      .catch(() => addToast("Failed to load Today", "error"))
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [addToast]);

  async function handleSubmit() {
    const trimmed = question.trim();
    if (trimmed.length < 12) {
      addToast("Ask with a little more context", "error");
      return;
    }
    setSubmitting(true);
    try {
      const res = await createAsk({
        question: trimmed,
        desired_help_type: helpType,
        urgency: helpType === "peer" ? "soon" : "this_week",
        visibility: "community",
      });
      setQuestion("");
      setData((prev) => ({
        latest_ask: res.ask,
        bridges: res.bridges,
        help_sessions: prev?.help_sessions ?? [],
        trust_profile: prev?.trust_profile,
        starter_prompts: prev?.starter_prompts ?? [],
      }));
      addToast("Human bridges found", "success");
    } catch {
      addToast("Failed to create ask", "error");
    } finally {
      setSubmitting(false);
    }
  }

  function updateBridge(updated: Bridge) {
    setData((prev) =>
      prev
        ? {
            ...prev,
            bridges: prev.bridges.map((b) => (b.id === updated.id ? updated : b)),
          }
        : prev,
    );
  }

  async function handleJoin(session: HelpSession) {
    try {
      const updated = await joinHelpSession(session.id);
      setData((prev) =>
        prev
          ? {
              ...prev,
              help_sessions: prev.help_sessions.map((s) =>
                s.id === updated.id ? updated : s,
              ),
            }
          : prev,
      );
      addToast("Joined session", "success");
    } catch {
      addToast("Failed to join session", "error");
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    );
  }

  const bridges = data?.bridges ?? [];
  const sessions = data?.help_sessions ?? [];
  const starterPrompts = data?.starter_prompts ?? [];

  return (
    <div className="space-y-6">
      <section className="pt-2">
        <p className="text-xs font-semibold uppercase tracking-[0.12em] text-[var(--color-text-muted)]">
          Human layer after AI
        </p>
        <h1 className="mt-2 text-2xl font-semibold leading-tight">
          Find the person who has lived your question.
        </h1>
      </section>

      <section className="rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-3">
        <textarea
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder="What do you need human perspective on?"
          className="min-h-28 w-full resize-none bg-transparent p-2 text-base outline-none placeholder:text-[var(--color-text-muted)]"
        />
        <div className="grid gap-2 sm:grid-cols-4">
          {quickIntents.map((intent) => (
            <button
              key={intent.type}
              type="button"
              onClick={() => setHelpType(intent.type)}
              className={`rounded-lg border px-3 py-2 text-left transition-colors ${
                helpType === intent.type
                  ? "border-[var(--color-primary)] bg-[var(--color-primary-subtle)]"
                  : "border-[var(--color-border)] hover:bg-[var(--color-surface)]"
              }`}
            >
              <span className="block text-sm font-medium">{intent.label}</span>
              <span className="mt-0.5 block text-[11px] leading-snug text-[var(--color-text-muted)]">
                {intent.helper}
              </span>
            </button>
          ))}
        </div>
        <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap gap-2">
            {starterPrompts.slice(0, 2).map((prompt) => (
              <button
                key={prompt}
                type="button"
                onClick={() => setQuestion(prompt)}
                className="rounded-full bg-[var(--color-surface)] px-3 py-1 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
              >
                {prompt}
              </button>
            ))}
          </div>
          <Button onClick={handleSubmit} loading={submitting}>
            Find humans
          </Button>
        </div>
      </section>

      {data?.latest_ask && (
        <section className="rounded-lg border border-[var(--color-border)] p-4">
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-[var(--color-text-muted)]">
            Current ask
          </p>
          <p className="mt-2 text-sm leading-relaxed">{data.latest_ask.question}</p>
          <p className="mt-2 text-xs text-[var(--color-text-muted)]">
            {data.latest_ask.triage_summary}
          </p>
        </section>
      )}

      <section className="space-y-3">
        <div className="flex items-end justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold">Human bridges</h2>
            <p className="text-xs text-[var(--color-text-muted)]">
              Quick, explainable routes to advice, peers, and perspective.
            </p>
          </div>
        </div>
        {bridges.length === 0 ? (
          <p className="rounded-lg border border-dashed border-[var(--color-border)] p-4 text-sm text-[var(--color-text-muted)]">
            Ask a question to generate your first bridges.
          </p>
        ) : (
          bridges.map((bridge) => (
            <BridgeCard
              key={bridge.id}
              bridge={bridge}
              onUpdate={updateBridge}
            />
          ))
        )}
      </section>

      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Live help rooms</h2>
        <div className="grid gap-3 sm:grid-cols-3">
          {sessions.map((session) => (
            <article
              key={session.id}
              className="rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4"
            >
              <p className="text-sm font-semibold">{session.title}</p>
              <p className="mt-2 min-h-10 text-xs leading-relaxed text-[var(--color-text-muted)]">
                {session.description}
              </p>
              <div className="mt-4 flex items-center justify-between gap-2">
                <span className="text-xs text-[var(--color-text-muted)]">
                  {session.member_count} inside
                </span>
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => handleJoin(session)}
                >
                  Join
                </Button>
              </div>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
