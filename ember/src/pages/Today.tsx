import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import type {
  AskedQuestion,
  AskVisibility,
  Bridge,
  DesiredHelpType,
  TodayResponse,
} from "@pulse/drift/types";
import {
  createAsk,
  getToday,
  respondBridge,
  updateAskVisibility,
} from "../api/advice";
import BridgeCard from "../components/advice/BridgeCard";
import AskPathCard from "../components/feed/AskPathCard";
import Button from "../components/ui/Button";
import Spinner from "../components/ui/Spinner";
import { usePageTitle } from "../hooks/usePageTitle";
import { useUiStore } from "../store/uiStore";

const quickIntents: Array<{ type: DesiredHelpType; label: string }> = [
  { type: "advice", label: "Advice" },
  { type: "peer", label: "Peer" },
  { type: "mentor", label: "Mentor" },
  { type: "feedback", label: "Feedback" },
];

const visibilityOptions: Array<{
  value: AskVisibility;
  label: string;
  helper: string;
}> = [
  { value: "private", label: "Private", helper: "Only people Pulse routes it to" },
  { value: "community", label: "Community", helper: "People whose context matches" },
  { value: "public", label: "Public", helper: "Anyone — can join the Commons" },
];

function QuestionAnswers({
  item,
  publishing,
  onPublish,
}: {
  item: AskedQuestion;
  publishing: boolean;
  onPublish: (askId: string, anonymous: boolean) => void;
}) {
  const answeredBridges = item.bridges.filter(
    (bridge) => (bridge.responses?.length ?? 0) > 0,
  );
  const routedCount = item.bridges.filter(
    (bridge) => bridge.status === "asked" || bridge.status === "responded",
  ).length;
  const isPublic = item.ask.visibility === "public";

  return (
    <article className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-text-muted)]">
            Your question
          </p>
          <p className="mt-1.5 text-sm font-medium leading-relaxed">
            {item.ask.question}
          </p>
        </div>
        <span className="shrink-0 rounded-full bg-[var(--color-surface)] px-2 py-1 text-[11px] text-[var(--color-text-muted)]">
          {item.answer_count > 0
            ? `${item.answer_count} answered`
            : `${routedCount} routed`}
        </span>
      </div>

      {answeredBridges.length > 0 ? (
        <div className="mt-4 space-y-3">
          {answeredBridges.map((bridge) =>
            (bridge.responses ?? []).map((response) => (
              <div
                key={response.id}
                className="rounded-[var(--radius-md)] bg-[var(--color-surface)] p-3"
              >
                <p className="text-xs font-semibold">
                  {response.responder?.display_name ||
                    response.responder?.handle ||
                    bridge.recommended_user?.display_name ||
                    bridge.recommended_user?.handle ||
                    "Someone"}
                </p>
                <p className="mt-1.5 whitespace-pre-wrap text-sm leading-relaxed text-[var(--color-text-secondary)]">
                  {response.body}
                </p>
              </div>
            )),
          )}
        </div>
      ) : (
        <p className="mt-3 text-xs leading-relaxed text-[var(--color-text-muted)]">
          No answer yet. Pulse already routed this to people whose context is
          close enough to help.
        </p>
      )}

      {answeredBridges.length > 0 && !isPublic && (
        <div className="mt-4 flex flex-wrap gap-2 border-t border-[var(--color-border)] pt-3">
          <Button
            size="sm"
            variant="secondary"
            onClick={() => onPublish(item.ask.id, true)}
            loading={publishing}
          >
            Add to Commons anonymously
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => onPublish(item.ask.id, false)}
            disabled={publishing}
          >
            Add with my name
          </Button>
        </div>
      )}
      {isPublic && (
        <Link
          to="/commons"
          className="mt-4 inline-flex text-xs font-medium text-[var(--color-accent)] hover:underline"
        >
          In Commons
        </Link>
      )}
    </article>
  );
}

export default function Today() {
  usePageTitle("Today");
  const [data, setData] = useState<TodayResponse | null>(null);
  const [question, setQuestion] = useState("");
  const [helpType, setHelpType] = useState<DesiredHelpType>("advice");
  const [visibility, setVisibility] = useState<AskVisibility>("community");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [responseDrafts, setResponseDrafts] = useState<Record<string, string>>(
    {},
  );
  const [publishing, setPublishing] = useState(false);
  const addToast = useUiStore((s) => s.addToast);

  useEffect(() => {
    let cancelled = false;
    getToday()
      .then((today) => {
        if (cancelled) return;
        setData(today);
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
        visibility,
      });
      setQuestion("");
      setData((prev) => ({
        latest_ask: res.ask,
        recent_asks: [
          {
            ask: res.ask,
            bridges: res.bridges,
            answer_count: res.bridges.reduce(
              (count, bridge) => count + (bridge.responses?.length ?? 0),
              0,
            ),
            last_at: res.ask.created_at,
          },
          ...(prev?.recent_asks ?? []),
        ],
        bridges: res.bridges,
        incoming_bridges: prev?.incoming_bridges ?? [],
        perspective_inbox: prev?.perspective_inbox ?? [],
        response_receipts: prev?.response_receipts ?? [],
        help_sessions: prev?.help_sessions ?? [],
        shared_rooms: prev?.shared_rooms ?? [],
        relationship_trails: prev?.relationship_trails ?? [],
        trust_profile: prev?.trust_profile,
        starter_prompts: prev?.starter_prompts ?? [],
      }));
      const routed = res.bridges.filter(
        (b) => b.status === "asked" || b.status === "responded",
      ).length;
      addToast(
        routed > 0
          ? `Routed to ${routed} ${routed === 1 ? "person" : "people"} who've lived this`
          : "We're finding people who've lived this",
        "success",
      );
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
            incoming_bridges: (prev.incoming_bridges ?? []).map((b) =>
              b.id === updated.id ? updated : b,
            ),
            perspective_inbox: (prev.perspective_inbox ?? [])
              .map((b) => (b.id === updated.id ? updated : b))
              .filter((b) => (b.responses?.length ?? 0) === 0),
            recent_asks: (prev.recent_asks ?? []).map((item) => ({
              ...item,
              bridges: item.bridges.map((b) =>
                b.id === updated.id ? updated : b,
              ),
              answer_count: item.bridges.reduce((count, b) => {
                const bridge = b.id === updated.id ? updated : b;
                return count + (bridge.responses?.length ?? 0);
              }, 0),
            })),
          }
        : prev,
    );
  }

  async function handleRespond(bridge: Bridge) {
    const message = (responseDrafts[bridge.id] ?? "").trim();
    if (message.length < 8) {
      addToast("Add a short perspective first", "error");
      return;
    }
    try {
      const updated = await respondBridge(bridge.id, message);
      updateBridge(updated);
      setResponseDrafts((prev) => ({ ...prev, [bridge.id]: "" }));
      addToast("Perspective offered", "success");
    } catch {
      addToast("Failed to offer perspective", "error");
    }
  }

  async function handlePublish(askId: string, anonymous: boolean) {
    setPublishing(true);
    try {
      const updated = await updateAskVisibility(askId, {
        visibility: "public",
        anonymous,
      });
      setData((prev) =>
        prev
          ? {
              ...prev,
              latest_ask:
                prev.latest_ask?.id === updated.id ? updated : prev.latest_ask,
              recent_asks: (prev.recent_asks ?? []).map((item) =>
                item.ask.id === updated.id ? { ...item, ask: updated } : item,
              ),
            }
          : prev,
      );
      addToast("Published to the Commons", "success");
    } catch {
      addToast("Failed to publish", "error");
    } finally {
      setPublishing(false);
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-16">
        <Spinner size="lg" />
      </div>
    );
  }

  const bridges = data?.bridges ?? [];
  const incomingBridges = data?.incoming_bridges ?? [];
  const perspectiveInbox = data?.perspective_inbox ?? [];
  const responseReceipts = data?.response_receipts ?? [];
  const relationshipTrails = data?.relationship_trails ?? [];
  const recentAsks = data?.recent_asks ?? [];
  const answeredAsks = recentAsks.filter((item) => item.answer_count > 0);
  const waitingAsks = recentAsks.filter(
    (item) => item.answer_count === 0 && item.bridges.length > 0,
  );
  const sentPerspectives = incomingBridges.filter(
    (bridge) => (bridge.responses?.length ?? 0) > 0,
  );
  const starterPrompts = data?.starter_prompts ?? [];
  const trustProfile = data?.trust_profile;
  const latestAsk = data?.latest_ask;
  const answeredCount = bridges.filter((b) => (b.responses?.length ?? 0) > 0)
    .length;

  return (
    <div className="space-y-8 pb-4">
      <section className="pt-4">
        <h1 className="text-[28px] font-semibold leading-[1.15] tracking-tight">
          Today
          <br />
          <span className="text-[var(--color-text-muted)]">
            answers, asks, and next people.
          </span>
        </h1>
        <p className="mt-3 text-sm leading-relaxed text-[var(--color-text-muted)]">
          This is the place for human loops: what came back to you, what needs
          your perspective, and who Pulse can connect next.
        </p>
      </section>

      <section className="grid gap-2 sm:grid-cols-4">
        {[
          {
            label: "Answers",
            value: answeredAsks.reduce((sum, item) => sum + item.answer_count, 0),
            to: "#answers",
          },
          { label: "For you", value: perspectiveInbox.length, to: "#inbox" },
          { label: "Waiting", value: waitingAsks.length, to: "#waiting" },
          { label: "Trails", value: relationshipTrails.length, to: "#trails" },
        ].map((step) => (
          <a
            key={step.label}
            href={step.to}
            className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] px-3 py-3 transition-colors hover:bg-[var(--color-surface)]"
          >
            <span className="block text-[11px] font-semibold uppercase tracking-widest text-[var(--color-text-muted)]">
              {step.label}
            </span>
            <span className="mt-1 block text-xl font-semibold">
              {step.value}
            </span>
          </a>
        ))}
      </section>

      <section id="answers" className="space-y-3 scroll-mt-20">
        <div className="flex items-end justify-between gap-4">
          <div>
            <h2 className="text-[17px] font-semibold">
              Answers to your questions
            </h2>
            <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
              Read them here first. Commons is optional.
            </p>
          </div>
          <Link
            to="/commons"
            className="shrink-0 text-xs font-medium text-[var(--color-accent)] hover:underline"
          >
            Moments in common
          </Link>
        </div>
        {answeredAsks.length > 0 ? (
          <div className="space-y-3">
            {answeredAsks.map((item) => (
              <QuestionAnswers
                key={item.ask.id}
                item={item}
                publishing={publishing}
                onPublish={handlePublish}
              />
            ))}
          </div>
        ) : (
          <div className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border)] p-5">
            <p className="text-sm text-[var(--color-text-muted)]">
              No answers yet. Ask a question below and Pulse will route it to
              people with nearby lived context.
            </p>
          </div>
        )}
      </section>

      <section className="rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4">
        <div className="mb-3 flex items-center justify-between gap-3">
          <div>
            <p className="text-sm font-semibold">Need a human read?</p>
            <p className="text-xs text-[var(--color-text-muted)]">
              Ask once. Pulse routes it quietly.
            </p>
          </div>
          <Link
            to="/upload"
            className="shrink-0 rounded-[var(--radius-sm)] bg-[var(--color-surface)] px-3 py-1.5 text-xs font-medium hover:bg-[var(--color-surface-hover)]"
          >
            Share moment
          </Link>
        </div>
        <textarea
          name="ask-question"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder="What do you need a human perspective on?"
          className="min-h-20 w-full resize-none bg-transparent p-1 text-[15px] leading-relaxed outline-none placeholder:text-[var(--color-text-muted)]"
        />

        {/* Intent chips */}
        <div className="mb-3 mt-1 flex gap-2">
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

        {/* Visibility selector */}
        <div className="mb-3">
          <div className="flex gap-1.5">
            {visibilityOptions.map((opt) => (
              <button
                key={opt.value}
                type="button"
                onClick={() => setVisibility(opt.value)}
                className={`flex-1 rounded-[var(--radius-sm)] border px-2 py-1.5 text-center transition-all ${
                  visibility === opt.value
                    ? "border-[var(--color-accent)] bg-[var(--color-accent-subtle)]"
                    : "border-[var(--color-border)] hover:border-[var(--color-border-emphasis)]"
                }`}
              >
                <span className="block text-[12px] font-medium">
                  {opt.label}
                </span>
              </button>
            ))}
          </div>
          <p className="mt-1.5 text-[11px] text-[var(--color-text-muted)]">
            {visibilityOptions.find((o) => o.value === visibility)?.helper}
          </p>
        </div>

        <div className="flex items-center justify-end">
          <Button
            size="sm"
            variant="accent"
            onClick={handleSubmit}
            loading={submitting}
          >
            Find people who&rsquo;ve lived this
          </Button>
        </div>
      </section>

      {/* Starter prompts — warm entry when there's no ask yet */}
      {!latestAsk && starterPrompts.length > 0 && (
        <section>
          <p className="mb-2 text-[11px] font-semibold uppercase tracking-widest text-[var(--color-text-muted)]">
            Not sure where to start?
          </p>
          <div className="flex flex-wrap gap-2">
            {starterPrompts.map((prompt) => (
              <button
                key={prompt}
                type="button"
                onClick={() => setQuestion(prompt)}
                className="rounded-full border border-[var(--color-border)] bg-[var(--color-bg-elevated)] px-3 py-1.5 text-xs text-[var(--color-text-secondary)] transition-colors hover:border-[var(--color-border-emphasis)] hover:text-[var(--color-text)]"
              >
                {prompt}
              </button>
            ))}
          </div>
        </section>
      )}

      {/* Perspective inbox — directed context that should not feel empty */}
      {perspectiveInbox.length > 0 && (
        <section id="inbox" className="space-y-3 scroll-mt-20">
          <div>
            <h2 className="text-[17px] font-semibold">Perspective inbox</h2>
            <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
              Open asks routed to you because your graph has nearby lived
              context.
            </p>
          </div>
          {perspectiveInbox.map((bridge) => (
            <AskPathCard
              key={bridge.id}
              bridge={bridge}
              onUpdate={updateBridge}
              compact
            />
          ))}
        </section>
      )}

      {waitingAsks.length > 0 && (
        <section id="waiting" className="space-y-3 scroll-mt-20">
          <div>
            <h2 className="text-[17px] font-semibold">Waiting on answers</h2>
            <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
              Questions already routed by Pulse. Nothing else to click.
            </p>
          </div>
          <div className="space-y-2">
            {waitingAsks.map((item) => {
              const routed = item.bridges.filter(
                (bridge) =>
                  bridge.status === "asked" || bridge.status === "responded",
              ).length;
              return (
                <article
                  key={item.ask.id}
                  className="rounded-[var(--radius-md)] bg-[var(--color-surface)] p-4"
                >
                  <p className="text-sm font-medium leading-relaxed">
                    {item.ask.question}
                  </p>
                  <p className="mt-2 text-xs text-[var(--color-text-muted)]">
                    Routed to {routed} {routed === 1 ? "person" : "people"}.
                    Their answers will appear at the top of Today.
                  </p>
                </article>
              );
            })}
          </div>
        </section>
      )}

      {/* Response receipts */}
      {responseReceipts.length > 0 && (
        <section className="space-y-3">
          <div>
            <h2 className="text-[17px] font-semibold">Response receipts</h2>
            <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
              Proof that your perspective landed, without turning it into likes.
            </p>
          </div>
          <div className="space-y-2">
            {responseReceipts.map((receipt) => (
              <article
                key={receipt.bridge_id}
                className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="text-sm font-semibold">
                      {receipt.user.display_name || receipt.user.handle}
                    </p>
                    <p className="mt-1 line-clamp-2 text-xs leading-relaxed text-[var(--color-text-muted)]">
                      {receipt.question}
                    </p>
                  </div>
                  <div className="flex shrink-0 flex-wrap justify-end gap-1">
                    {receipt.signals.map((signal) => (
                      <span
                        key={signal}
                        className="rounded-full bg-[var(--color-surface)] px-2 py-1 text-[11px] font-medium text-[var(--color-success)]"
                      >
                        {signal.replace("_", " ")}
                      </span>
                    ))}
                  </div>
                </div>
              </article>
            ))}
          </div>
        </section>
      )}

      {sentPerspectives.length > 0 && (
        <section className="space-y-3">
          <div>
            <h2 className="text-[17px] font-semibold">
              Perspectives you sent
            </h2>
            <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
              A lightweight history of where you helped.
            </p>
          </div>
          <div className="space-y-2">
            {sentPerspectives.map((bridge) => (
              <article
                key={bridge.id}
                className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4"
              >
                <p className="text-sm font-medium leading-relaxed">
                  {bridge.ask?.question}
                </p>
                <p className="mt-2 line-clamp-3 whitespace-pre-wrap text-xs leading-relaxed text-[var(--color-text-muted)]">
                  {bridge.responses?.[0]?.body}
                </p>
              </article>
            ))}
          </div>
        </section>
      )}

      {bridges.length > 0 && answeredCount === 0 && (
        <section className="space-y-3">
          <div>
            <h2 className="text-[17px] font-semibold">People matched to you</h2>
            <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
              Routed automatically by lived experience. Reach more if you like.
            </p>
          </div>
          {bridges.map((bridge) => (
            <BridgeCard key={bridge.id} bridge={bridge} onUpdate={updateBridge} />
          ))}
        </section>
      )}

      {/* Asked of you — directed incoming only */}
      {incomingBridges.length > 0 && (
        <section className="space-y-3">
          <div>
            <h2 className="text-[17px] font-semibold">Asked of you</h2>
            <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
              Pulse routed these here because you&rsquo;ve lived something close.
              Share the one thing you&rsquo;d tell them.
            </p>
          </div>
          {incomingBridges.map((bridge) => {
            const asker = bridge.ask?.user;
            const response = bridge.responses?.[0];
            return (
              <article
                key={bridge.id}
                className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4"
              >
                <div className="flex items-start gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--color-surface-active)] text-sm font-semibold text-[var(--color-text-secondary)]">
                    {asker?.display_name?.[0] || asker?.handle?.[0] || "?"}
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-semibold">
                      {asker?.display_name || asker?.handle || "Someone"}
                    </p>
                    <p className="mt-2 text-sm leading-relaxed">
                      {bridge.ask?.question}
                    </p>
                    <p className="mt-1.5 text-xs leading-relaxed text-[var(--color-text-muted)]">
                      {bridge.reason}
                    </p>
                  </div>
                </div>
                {response ? (
                  <div className="mt-4 rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
                    <p className="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-text-muted)]">
                      Your perspective
                    </p>
                    <p className="mt-2 text-sm leading-relaxed whitespace-pre-wrap">
                      {response.body}
                    </p>
                  </div>
                ) : (
                  <div className="mt-4 space-y-3">
                    <textarea
                      name={`perspective-${bridge.id}`}
                      value={responseDrafts[bridge.id] ?? ""}
                      onChange={(e) =>
                        setResponseDrafts((prev) => ({
                          ...prev,
                          [bridge.id]: e.target.value,
                        }))
                      }
                      placeholder="Share the one thing you would tell them…"
                      maxLength={1200}
                      className="min-h-20 w-full resize-none rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg)] p-3 text-sm leading-relaxed outline-none placeholder:text-[var(--color-text-muted)] focus:border-[var(--color-accent)]"
                    />
                    <div className="flex justify-end">
                      <Button
                        size="sm"
                        variant="accent"
                        onClick={() => handleRespond(bridge)}
                      >
                        Offer perspective
                      </Button>
                    </div>
                  </div>
                )}
              </article>
            );
          })}
        </section>
      )}

      {/* Paths as relationship trails */}
      {relationshipTrails.length > 0 && (
        <section id="trails" className="space-y-3 scroll-mt-20">
          <div>
            <h2 className="text-[17px] font-semibold">
              Relationship trails
            </h2>
            <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
              Paths followed by you or people close to you, so old moments can
              keep carrying context.
            </p>
          </div>
          <div className="space-y-2">
            {relationshipTrails.map((path) => (
              <Link
                key={path.id}
                to={`/paths/${path.id}`}
                className="flex items-center justify-between gap-4 rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4 transition-colors hover:bg-[var(--color-surface)]"
              >
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold">
                    {path.title}
                  </p>
                  <p className="mt-0.5 truncate text-xs text-[var(--color-text-muted)]">
                    {path.items?.length ?? 0} moments by{" "}
                    {path.creator?.display_name || path.creator?.handle}
                  </p>
                </div>
                <span className="shrink-0 text-xs font-medium text-[var(--color-accent)]">
                  Open
                </span>
              </Link>
            ))}
          </div>
        </section>
      )}

      {/* First-run empty state */}
      {bridges.length === 0 && !latestAsk && incomingBridges.length === 0 && (
        <section className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border)] p-6 text-center">
          <p className="text-sm text-[var(--color-text-muted)]">
            Ask your first question above. Pulse finds the people who&rsquo;ve
            lived it — no feed to scroll, no audience to perform for.
          </p>
        </section>
      )}

      {/* Offer help — the other side of the graph */}
      <section>
        {trustProfile ? (
          <div className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-text-muted)]">
                  You can help with
                </p>
                <p className="mt-1.5 text-sm font-medium leading-snug">
                  {trustProfile.topics || "Add the topics you can speak to"}
                </p>
              </div>
              <Link
                to="/settings"
                className="shrink-0 text-xs font-medium text-[var(--color-accent)] hover:underline"
              >
                Edit
              </Link>
            </div>
          </div>
        ) : (
          <Link
            to="/settings"
            className="group flex items-center justify-between gap-4 rounded-[var(--radius-md)] border border-dashed border-[var(--color-border-emphasis)] p-4 transition-colors hover:bg-[var(--color-bg-elevated)]"
          >
            <div className="min-w-0">
              <p className="text-sm font-semibold">Be someone others can reach</p>
              <p className="mt-0.5 text-xs leading-relaxed text-[var(--color-text-muted)]">
                Tell Pulse what you&rsquo;ve lived through, and it&rsquo;ll route
                the right people to you.
              </p>
            </div>
            <span className="shrink-0 text-[var(--color-text-muted)] transition-colors group-hover:text-[var(--color-text)]">
              <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <line x1="5" y1="12" x2="19" y2="12" />
                <polyline points="12 5 19 12 12 19" />
              </svg>
            </span>
          </Link>
        )}
      </section>

    </div>
  );
}
