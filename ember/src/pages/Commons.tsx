import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import type {
  AffinityFeedItem,
  CommonsEntry,
  FeedMoment,
} from "@pulse/drift/types";
import { addPerspective, listCommons } from "../api/advice";
import { getFeed } from "../api/content";
import ContentModal from "../components/feed/ContentModal";
import MediaFallback from "../components/ui/MediaFallback";
import Button from "../components/ui/Button";
import Spinner from "../components/ui/Spinner";
import { usePageTitle } from "../hooks/usePageTitle";
import { useUiStore } from "../store/uiStore";

function Avatar({ char }: { char: string }) {
  return (
    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-[var(--color-surface-active)] text-sm font-semibold text-[var(--color-text-secondary)]">
      {char}
    </div>
  );
}

function MomentTile({
  item,
  onClick,
}: {
  item: AffinityFeedItem;
  onClick: () => void;
}) {
  const moment = item.content!;
  return (
    <button
      type="button"
      onClick={onClick}
      className="group min-w-[72%] snap-start overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] text-left sm:min-w-[46%]"
    >
      {moment.content_type === "image" && (
        <MediaFallback
          src={moment.media_url}
          alt={moment.body || "Moment"}
          type="image"
          className="aspect-[4/3] w-full object-cover"
          loading="lazy"
        />
      )}
      {(moment.content_type === "video" ||
        moment.content_type === "short_video") && (
        <MediaFallback
          src={moment.media_url}
          alt={moment.body || "Moment"}
          type={moment.content_type}
          className="aspect-[4/3] w-full object-cover bg-black"
          muted
          playsInline
          preload="metadata"
        />
      )}
      {moment.content_type === "text" && (
        <div className="flex aspect-[4/3] items-center bg-[var(--color-surface)] p-4">
          <p className="line-clamp-5 text-sm leading-relaxed text-[var(--color-text-secondary)]">
            {moment.body}
          </p>
        </div>
      )}
      <div className="p-3">
        {(item.path_hint || item.reason) && (
          <div className="mb-2 flex flex-wrap items-center gap-1.5">
            {item.path_hint && (
              <span className="rounded-full bg-[var(--color-surface)] px-2 py-0.5 text-[10px] font-medium text-[var(--color-text-secondary)]">
                {item.path_hint}
              </span>
            )}
            {item.reason && (
              <span className="line-clamp-1 text-[11px] text-[var(--color-text-muted)]">
                {item.reason}
              </span>
            )}
          </div>
        )}
        <div className="flex items-center gap-2">
          <span className="flex h-6 w-6 items-center justify-center rounded-full bg-[var(--color-surface-active)] text-[10px] font-semibold text-[var(--color-text-secondary)]">
            {moment.creator?.display_name?.[0] || "?"}
          </span>
          <p className="truncate text-xs font-medium">
            {moment.creator?.display_name || moment.creator?.handle || "Someone"}
          </p>
        </div>
        {moment.tags?.length > 0 && (
          <p className="mt-2 truncate text-xs text-[var(--color-text-muted)]">
            {moment.tags.slice(0, 3).map((tag) => `#${tag.name}`).join(" ")}
          </p>
        )}
      </div>
    </button>
  );
}

function CommonsCard({ entry }: { entry: CommonsEntry }) {
  const { ask } = entry;
  const responses = entry.responses ?? [];
  const anonymous = ask.anonymous;
  const asker = ask.user;
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [added, setAdded] = useState(false);
  const addToast = useUiStore((s) => s.addToast);
  const answered = responses.length > 0;

  async function handleAdd() {
    const message = draft.trim();
    if (message.length < 8) {
      addToast("Share a little more of your perspective", "error");
      return;
    }
    setBusy(true);
    try {
      await addPerspective(ask.id, message);
      setAdded(true);
      setOpen(false);
      setDraft("");
      addToast("Your perspective was added", "success");
    } catch {
      addToast("Failed to add perspective", "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <article className="rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4">
      {/* Asker */}
      <div className="flex items-center gap-2.5">
        <Avatar char={anonymous ? "·" : asker?.display_name?.[0] || "?"} />
        <div className="min-w-0">
          <p className="text-sm font-medium">
            {anonymous ? "Someone" : asker?.display_name || asker?.handle}
            <span className="ml-1.5 font-normal text-[var(--color-text-muted)]">
              {answered ? "asked" : "is asking"}
            </span>
          </p>
          {ask.topic && (
            <p className="text-xs text-[var(--color-text-muted)]">
              on {ask.topic}
            </p>
          )}
        </div>
      </div>

      {/* Question */}
      <p className="mt-3 text-[15px] font-medium leading-relaxed">
        {ask.question}
      </p>

      {/* Perspectives */}
      {answered ? (
        <div className="mt-4 space-y-3">
          {responses.map((r) => (
            <div
              key={r.id}
              className="rounded-[var(--radius-md)] bg-[var(--color-surface)] p-3"
            >
              <div className="flex items-center gap-2">
                {r.responder ? (
                  <Link
                    to={`/profile/${r.responder.id}`}
                    className="text-xs font-semibold hover:text-[var(--color-accent)]"
                  >
                    {r.responder.display_name || r.responder.handle}
                  </Link>
                ) : (
                  <span className="text-xs font-semibold">Someone</span>
                )}
                <span className="text-[11px] text-[var(--color-text-muted)]">
                  shared their perspective
                </span>
              </div>
              <p className="mt-1.5 whitespace-pre-wrap text-sm leading-relaxed text-[var(--color-text-secondary)]">
                {r.body}
              </p>
            </div>
          ))}
        </div>
      ) : (
        <div className="mt-4 rounded-[var(--radius-md)] border border-dashed border-[var(--color-border-emphasis)] p-3">
          <p className="text-xs font-medium text-[var(--color-text-muted)]">
            Open question. Add the perspective you wish you had when you lived
            something close.
          </p>
        </div>
      )}

      {/* Add perspective — the non-Twitter alternative to a reply */}
      <div className="mt-3 border-t border-[var(--color-border)] pt-3">
        {added ? (
          <p className="text-xs font-medium text-[var(--color-success)]">
            ✓ Your perspective was added
          </p>
        ) : open ? (
          <div className="space-y-2">
            <textarea
              name={`commons-perspective-${ask.id}`}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              autoFocus
              maxLength={1200}
              placeholder="What did living this teach you?"
              className="min-h-20 w-full resize-none rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg)] p-3 text-sm leading-relaxed outline-none placeholder:text-[var(--color-text-muted)] focus:border-[var(--color-accent)]"
            />
            <div className="flex justify-end gap-2">
              <Button
                size="sm"
                variant="ghost"
                onClick={() => setOpen(false)}
                disabled={busy}
              >
                Cancel
              </Button>
              <Button
                size="sm"
                variant="accent"
                onClick={handleAdd}
                loading={busy}
              >
                Add perspective
              </Button>
            </div>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setOpen(true)}
            className="text-xs font-medium text-[var(--color-text-muted)] transition-colors hover:text-[var(--color-text)]"
          >
            I&rsquo;ve lived this too → add your perspective
          </button>
        )}
      </div>
    </article>
  );
}

export default function Commons() {
  usePageTitle("Commons");
  const [entries, setEntries] = useState<CommonsEntry[] | null>(null);
  const [moments, setMoments] = useState<AffinityFeedItem[]>([]);
  const [selectedMoment, setSelectedMoment] = useState<FeedMoment | null>(null);
  const [cursor, setCursor] = useState("");
  const [hasMore, setHasMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const addToast = useUiStore((s) => s.addToast);

  useEffect(() => {
    let cancelled = false;
    Promise.all([listCommons(), getFeed(undefined, 10)])
      .then(([res, feed]) => {
        if (cancelled) return;
        setEntries(res.items);
        setCursor(res.next_cursor);
        setHasMore(res.has_more);
        setMoments(
          feed.items
            .filter((item) => item.content)
            .map((item) => ({
              ...item,
              content: {
                ...item.content!,
                room_context: item.room_context,
              },
            })),
        );
      })
      .catch(() => {
        if (!cancelled) {
          setEntries([]);
          addToast("Failed to load the Commons", "error");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [addToast]);

  async function loadMore() {
    if (loadingMore || !hasMore || !cursor) return;
    setLoadingMore(true);
    try {
      const res = await listCommons(cursor);
      setEntries((prev) => [...(prev ?? []), ...res.items]);
      setCursor(res.next_cursor);
      setHasMore(res.has_more);
    } catch {
      addToast("Failed to load more", "error");
    } finally {
      setLoadingMore(false);
    }
  }

  if (entries === null) {
    return (
      <div className="flex justify-center py-16">
        <Spinner size="lg" />
      </div>
    );
  }

  const openEntries = entries.filter(
    (entry) => (entry.responses?.length ?? 0) === 0,
  );
  const answeredEntries = entries.filter(
    (entry) => (entry.responses?.length ?? 0) > 0,
  );

  return (
    <div className="space-y-5 pb-4">
      <section className="pt-4">
        <h1 className="text-[28px] font-semibold tracking-tight">Commons</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Shared context: open questions, lived answers, and moments people are
          orbiting right now.
        </p>
      </section>

      {moments.length > 0 && (
        <section className="space-y-3">
          <div>
            <h2 className="text-[17px] font-semibold">Moments in common</h2>
            <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
              Proof of what people are making, feeling, and returning to.
            </p>
          </div>
          <div className="-mx-4 flex snap-x gap-3 overflow-x-auto px-4 pb-1">
            {moments.map((moment) => (
              <MomentTile
                key={moment.id}
                item={moment}
                onClick={() => moment.content && setSelectedMoment(moment.content)}
              />
            ))}
          </div>
        </section>
      )}

      {entries.length === 0 ? (
        <section className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border)] p-8 text-center">
          <p className="text-sm text-[var(--color-text-muted)]">
            Commons is warming up. Public asks and shared moments will appear
            here as the graph gets signals.
          </p>
        </section>
      ) : (
        <section className="space-y-6">
          {openEntries.length > 0 && (
            <div className="space-y-3">
              <h2 className="text-[17px] font-semibold">Open questions</h2>
              {openEntries.map((entry) => (
                <CommonsCard key={entry.ask.id} entry={entry} />
              ))}
            </div>
          )}
          {answeredEntries.length > 0 && (
            <div className="space-y-3">
              <h2 className="text-[17px] font-semibold">Lived answers</h2>
              {answeredEntries.map((entry) => (
                <CommonsCard key={entry.ask.id} entry={entry} />
              ))}
            </div>
          )}
          {hasMore && (
            <div className="flex justify-center pt-2">
              <Button
                size="sm"
                variant="secondary"
                onClick={loadMore}
                loading={loadingMore}
              >
                Load more
              </Button>
            </div>
          )}
        </section>
      )}
      {selectedMoment && (
        <ContentModal
          content={selectedMoment}
          onClose={() => setSelectedMoment(null)}
        />
      )}
    </div>
  );
}
