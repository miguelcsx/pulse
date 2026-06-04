import { useEffect, useState } from "react";
import type {
  Bridge,
  DesiredHelpType,
  TodayResponse,
} from "@pulse/drift/types";
import { createAsk, getToday } from "../api/advice";
import BridgeCard from "../components/advice/BridgeCard";
import Button from "../components/ui/Button";
import Spinner from "../components/ui/Spinner";
import { usePageTitle } from "../hooks/usePageTitle";
import { useUiStore } from "../store/uiStore";

const quickIntents: Array<{
  type: DesiredHelpType;
  label: string;
  helper: string;
}> = [
  { type: "advice", label: "Advice", helper: "Lived context" },
  { type: "peer", label: "Peer", helper: "Shared stage" },
  { type: "mentor", label: "Mentor", helper: "Steps ahead" },
  { type: "feedback", label: "Feedback", helper: "Fresh eyes" },
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
      addToast("Add a little more context to your question", "error");
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
      addToast("Bridges found", "success");
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
            bridges: prev.bridges.map((b) =>
              b.id === updated.id ? updated : b,
            ),
          }
        : prev,
    );
  }

  if (loading) {
    return (
      <div className="flex justify-center py-16">
        <Spinner size="lg" />
      </div>
    );
  }

  const bridges = data?.bridges ?? [];
  const starterPrompts = data?.starter_prompts ?? [];

  return (
    <div className="space-y-8 pb-4">
      {/* Hero */}
      <section className="pt-4">
        <h1 className="text-[28px] font-semibold leading-[1.15] tracking-tight">
          Find a person who&rsquo;s
          <br />
          <span className="text-[var(--color-text-muted)]">
            lived what you&rsquo;re facing.
          </span>
        </h1>
        <p className="mt-3 text-sm leading-relaxed text-[var(--color-text-muted)]">
          Ask for perspective or share moments. Pulse turns those signals into
          bridges with people who share context, taste, or lived experience.
        </p>
      </section>

      {/* Ask box */}
      <section className="rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4">
        <textarea
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder="What do you need perspective on?"
          className="min-h-20 w-full resize-none bg-transparent p-1 text-[15px] leading-relaxed outline-none placeholder:text-[var(--color-text-muted)]"
        />

        {/* Intent chips */}
        <div className="flex gap-2 mt-1 mb-3">
          {quickIntents.map((intent) => (
            <button
              key={intent.type}
              type="button"
              onClick={() => setHelpType(intent.type)}
              className={`rounded-full px-3 py-1.5 text-[13px] font-medium transition-all ${
                helpType === intent.type
                  ? "bg-[var(--color-primary)] text-[var(--color-bg)]"
                  : "bg-[var(--color-surface)] text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
              }`}
            >
              {intent.label}
            </button>
          ))}
        </div>

        {/* Starters + submit */}
        <div className="flex items-center justify-between gap-3">
          <div className="flex gap-2 overflow-x-auto">
            {starterPrompts.slice(0, 2).map((prompt) => (
              <button
                key={prompt}
                type="button"
                onClick={() => setQuestion(prompt)}
                className="shrink-0 rounded-full bg-[var(--color-surface)] px-3 py-1.5 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
              >
                {prompt}
              </button>
            ))}
          </div>
          <Button
            size="sm"
            variant="accent"
            onClick={handleSubmit}
            loading={submitting}
            className="shrink-0"
          >
            Find
          </Button>
        </div>
      </section>

      {/* Current ask */}
      {data?.latest_ask && (
        <section className="rounded-[var(--radius-md)] bg-[var(--color-surface)] p-4">
          <p className="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-text-muted)] mb-2">
            Your ask
          </p>
          <p className="text-sm leading-relaxed">{data.latest_ask.question}</p>
          {data.latest_ask.triage_summary && (
            <p className="mt-2 text-xs text-[var(--color-text-muted)]">
              {data.latest_ask.triage_summary}
            </p>
          )}
        </section>
      )}

      {/* Bridges */}
      {bridges.length > 0 && (
        <section className="space-y-3">
          <div>
            <h2 className="text-[17px] font-semibold">Bridges</h2>
            <p className="text-xs text-[var(--color-text-muted)] mt-0.5">
              People matched by lived experience
            </p>
          </div>
          {bridges.map((bridge) => (
            <BridgeCard
              key={bridge.id}
              bridge={bridge}
              onUpdate={updateBridge}
            />
          ))}
        </section>
      )}

      {/* Empty state for bridges */}
      {bridges.length === 0 && !data?.latest_ask && (
        <section className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border)] p-6 text-center">
          <p className="text-sm text-[var(--color-text-muted)]">
            Ask a question above to find your first human bridges.
          </p>
        </section>
      )}

    </div>
  );
}
