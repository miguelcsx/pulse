import { Fragment, useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import type {
  AffinityFeedItem,
  AskVisibility,
  Bridge,
  DesiredHelpType,
  FeedMoment,
  TodayResponse,
} from "@pulse/drift/types";
import {
  createAsk,
  getToday,
  respondBridge,
  updateAskVisibility,
} from "../api/advice";
import { getFeed } from "../api/content";
import BridgeCard from "../components/advice/BridgeCard";
import AskPathCard from "../components/feed/AskPathCard";
import ContentModal from "../components/feed/ContentModal";
import FeedCard from "../components/feed/FeedCard";
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

export default function Today() {
  usePageTitle("Today");
  const [data, setData] = useState<TodayResponse | null>(null);
  const [pathItems, setPathItems] = useState<AffinityFeedItem[]>([]);
  const [pathCursor, setPathCursor] = useState("");
  const [pathHasMore, setPathHasMore] = useState(false);
  const [pathLoadingMore, setPathLoadingMore] = useState(false);
  const [selectedMoment, setSelectedMoment] = useState<FeedMoment | null>(null);
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

  const loadPath = useCallback(
    async (cursor?: string) => {
      const res = await getFeed(cursor, 12);
      setPathItems((prev) => (cursor ? [...prev, ...res.items] : res.items));
      setPathCursor(res.next_cursor);
      setPathHasMore(res.has_more);
    },
    [],
  );

  useEffect(() => {
    let cancelled = false;
    Promise.all([getToday(), getFeed(undefined, 12)])
      .then(([today, feed]) => {
        if (cancelled) return;
        setData(today);
        setPathItems(feed.items);
        setPathCursor(feed.next_cursor);
        setPathHasMore(feed.has_more);
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
      loadPath().catch(() => {});
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
          }
        : prev,
    );
    setPathItems((prev) =>
      prev.map((item) =>
        item.bridge?.id === updated.id ? { ...item, bridge: updated } : item,
      ),
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

  async function handlePublish(anonymous: boolean) {
    if (!data?.latest_ask) return;
    setPublishing(true);
    try {
      const updated = await updateAskVisibility(data.latest_ask.id, {
        visibility: "public",
        anonymous,
      });
      setData((prev) => (prev ? { ...prev, latest_ask: updated } : prev));
      addToast("Published to the Commons", "success");
    } catch {
      addToast("Failed to publish", "error");
    } finally {
      setPublishing(false);
    }
  }

  async function loadMorePath() {
    if (pathLoadingMore || !pathHasMore || !pathCursor) return;
    setPathLoadingMore(true);
    try {
      await loadPath(pathCursor);
    } catch {
      addToast("Failed to load more path items", "error");
    } finally {
      setPathLoadingMore(false);
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
  const sharedRooms = data?.shared_rooms ?? [];
  const relationshipTrails = data?.relationship_trails ?? [];
  const starterPrompts = data?.starter_prompts ?? [];
  const trustProfile = data?.trust_profile;
  const latestAsk = data?.latest_ask;
  const routedCount = bridges.filter(
    (b) => b.status === "asked" || b.status === "responded",
  ).length;
  const answeredCount = bridges.filter((b) => (b.responses?.length ?? 0) > 0)
    .length;
  const isPublished = latestAsk?.visibility === "public";

  return (
    <div className="space-y-8 pb-4">
      {/* Hero */}
      <section className="pt-4">
        <h1 className="text-[28px] font-semibold leading-[1.15] tracking-tight">
          Your affinity path
          <br />
          <span className="text-[var(--color-text-muted)]">
            through moments and people.
          </span>
        </h1>
        <p className="mt-3 text-sm leading-relaxed text-[var(--color-text-muted)]">
          Pulse routes the next useful thing: a moment, a person, or a question
          where your context can help.
        </p>
      </section>

      {/* Affinity itinerary */}
      <section className="grid gap-2 sm:grid-cols-4">
        {[
          { label: "Moments", value: pathItems.length, to: "#path" },
          { label: "Inbox", value: perspectiveInbox.length, to: "#inbox" },
          { label: "Rooms", value: sharedRooms.length, to: "#rooms" },
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

      {/* Ask box */}
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

      {/* Live affinity path */}
      <section id="path" className="space-y-3 scroll-mt-20">
        <div className="flex items-end justify-between gap-4">
          <div>
            <h2 className="text-[17px] font-semibold">Next in your path</h2>
            <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
              Mixed by affinity, recency, and diversity — not popularity.
            </p>
          </div>
          <Link
            to="/commons"
            className="shrink-0 text-xs font-medium text-[var(--color-accent)] hover:underline"
          >
            Commons
          </Link>
        </div>

        {pathItems.length === 0 ? (
          <div className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border)] p-6 text-center">
            <p className="text-sm text-[var(--color-text-muted)]">
              Your path is warming up. Share a tagged moment or ask for
              perspective so Pulse can find stronger context.
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {pathItems.map((item) => {
              const moment = item.content
                ? { ...item.content, room_context: item.room_context }
                : null;

              return (
                <Fragment key={`${item.unit_type}-${item.id}`}>
                  {item.unit_type === "ask" && item.bridge ? (
                    <AskPathCard
                      bridge={item.bridge}
                      onUpdate={updateBridge}
                      compact
                    />
                  ) : moment ? (
                    <div className="space-y-2">
                      {(item.path_hint || item.reason) && (
                        <div className="flex flex-wrap items-center gap-2 px-1 text-xs text-[var(--color-text-muted)]">
                          {item.path_hint && (
                            <span className="rounded-full bg-[var(--color-surface)] px-2 py-1 font-medium text-[var(--color-text-secondary)]">
                              {item.path_hint}
                            </span>
                          )}
                          {item.reason && <span>{item.reason}</span>}
                        </div>
                      )}
                      <FeedCard
                        content={moment}
                        onClick={() => setSelectedMoment(moment)}
                      />
                    </div>
                  ) : null}
                </Fragment>
              );
            })}
            {pathHasMore && (
              <div className="flex justify-center pt-1">
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={loadMorePath}
                  loading={pathLoadingMore}
                >
                  Continue path
                </Button>
              </div>
            )}
          </div>
        )}
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

      {/* Your ask + routing status */}
      {latestAsk && (
        <section className="space-y-3">
          <div className="rounded-[var(--radius-md)] bg-[var(--color-surface)] p-4">
            <p className="mb-2 text-[11px] font-semibold uppercase tracking-widest text-[var(--color-text-muted)]">
              Your ask
            </p>
            <p className="text-sm leading-relaxed">{latestAsk.question}</p>
            <p className="mt-3 flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
              <span className="h-1.5 w-1.5 rounded-full bg-[var(--color-success)]" />
              {answeredCount > 0
                ? `${answeredCount} ${answeredCount === 1 ? "person has" : "people have"} answered`
                : routedCount > 0
                  ? `Routed to ${routedCount} ${routedCount === 1 ? "person" : "people"} — their perspective lands here`
                  : "Finding people who've lived this…"}
            </p>
          </div>

          {/* Publish to Commons once answered */}
          {answeredCount > 0 && !isPublished && (
            <div className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border-emphasis)] p-4">
              <p className="text-sm font-medium">Help the next person</p>
              <p className="mt-0.5 text-xs leading-relaxed text-[var(--color-text-muted)]">
                Publish this ask and its perspectives to the Commons so others
                facing the same thing can find it.
              </p>
              <div className="mt-3 flex flex-wrap gap-2">
                <Button
                  size="sm"
                  variant="primary"
                  onClick={() => handlePublish(true)}
                  loading={publishing}
                >
                  Publish anonymously
                </Button>
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => handlePublish(false)}
                  disabled={publishing}
                >
                  Publish with my name
                </Button>
              </div>
            </div>
          )}
          {isPublished && (
            <Link
              to="/commons"
              className="flex items-center justify-between rounded-[var(--radius-md)] bg-[var(--color-accent-subtle)] px-4 py-3 text-sm font-medium text-[var(--color-accent)]"
            >
              <span>✓ Published to the Commons</span>
              <span className="text-xs">View →</span>
            </Link>
          )}
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

      {/* Bridges — the matched people + their answers */}
      {bridges.length > 0 && (
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

      {/* Shared context rooms */}
      {sharedRooms.length > 0 && (
        <section id="rooms" className="space-y-3 scroll-mt-20">
          <div>
            <h2 className="text-[17px] font-semibold">Shared context rooms</h2>
            <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
              Temporary rooms where the graph has enough nearby people to make
              the next answer easier.
            </p>
          </div>
          <div className="grid gap-2 sm:grid-cols-2">
            {sharedRooms.map((room) => (
              <div
                key={room.id}
                className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4 transition-colors hover:bg-[var(--color-surface)]"
              >
                <p className="text-sm font-semibold">{room.title}</p>
                <p className="mt-1 line-clamp-2 text-xs leading-relaxed text-[var(--color-text-muted)]">
                  {room.description}
                </p>
                <p className="mt-3 text-xs font-medium text-[var(--color-accent)]">
                  {room.member_count} nearby
                </p>
              </div>
            ))}
          </div>
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

      {selectedMoment && (
        <ContentModal
          content={selectedMoment}
          onClose={() => setSelectedMoment(null)}
        />
      )}
    </div>
  );
}
