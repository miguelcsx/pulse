import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import type { CommonsEntry } from "@pulse/drift/types";
import { addPerspective, listCommons } from "../api/advice";
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

function CommonsCard({ entry }: { entry: CommonsEntry }) {
  const { ask, responses } = entry;
  const anonymous = ask.anonymous;
  const asker = ask.user;
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [added, setAdded] = useState(false);
  const addToast = useUiStore((s) => s.addToast);

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
              asked
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
            <p className="mt-1.5 text-sm leading-relaxed whitespace-pre-wrap text-[var(--color-text-secondary)]">
              {r.body}
            </p>
          </div>
        ))}
      </div>

      {/* Add perspective — the non-Twitter alternative to a reply */}
      <div className="mt-3 border-t border-[var(--color-border)] pt-3">
        {added ? (
          <p className="text-xs font-medium text-[var(--color-success)]">
            ✓ Your perspective was added
          </p>
        ) : open ? (
          <div className="space-y-2">
            <textarea
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
  const [cursor, setCursor] = useState("");
  const [hasMore, setHasMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const addToast = useUiStore((s) => s.addToast);

  useEffect(() => {
    let cancelled = false;
    listCommons()
      .then((res) => {
        if (cancelled) return;
        setEntries(res.items);
        setCursor(res.next_cursor);
        setHasMore(res.has_more);
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

  return (
    <div className="space-y-5 pb-4">
      <section className="pt-4">
        <h1 className="text-[28px] font-semibold tracking-tight">Commons</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Real questions, answered by people who&rsquo;ve lived them. Published
          here so the next person doesn&rsquo;t have to ask alone.
        </p>
      </section>

      {entries.length === 0 ? (
        <section className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border)] p-8 text-center">
          <p className="text-sm text-[var(--color-text-muted)]">
            Nothing published yet. When an ask gets a great answer, its author
            can publish it here to help others.
          </p>
        </section>
      ) : (
        <section className="space-y-4">
          {entries.map((entry) => (
            <CommonsCard key={entry.ask.id} entry={entry} />
          ))}
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
    </div>
  );
}
