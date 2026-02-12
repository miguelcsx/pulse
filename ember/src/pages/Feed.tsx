import { useState, useEffect, useCallback, useRef, Fragment } from "react";
import { getFeed } from "../api/content";
import FeedCard from "../components/feed/FeedCard";
import ContentModal from "../components/feed/ContentModal";
import InlineSuggestionGroup from "../components/feed/InlineSuggestionGroup";
import Spinner from "../components/ui/Spinner";
import TrendingTags from "../components/content/TrendingTags";
import { useUiStore } from "../store/uiStore";
import { useFeedContextStore } from "../store/feedContextStore";
import { useVisibleRoomContext } from "../hooks/useVisibleRoomContext";
import { usePageTitle } from "../hooks/usePageTitle";
import type { FeedItem, Suggestion } from "@pulse/drift/types";

export default function Feed() {
  usePageTitle("Feed");
  const [items, setItems] = useState<FeedItem[]>([]);
  const [cursor, setCursor] = useState("");
  const [hasMore, setHasMore] = useState(true);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [selected, setSelected] = useState<FeedItem | null>(null);
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [error, setError] = useState<string | null>(null);
  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const loadingMoreRef = useRef(false);
  const addToast = useUiStore((s) => s.addToast);
  const setActiveRoom = useFeedContextStore((s) => s.setActiveRoom);
  const observe = useVisibleRoomContext(items);

  const loadFeed = useCallback(
    async (nextCursor?: string) => {
      try {
        setError(null);
        const res = await getFeed(nextCursor);
        if (nextCursor) {
          setItems((prev) => [...prev, ...res.items]);
        } else {
          setItems(res.items);
          if (res.suggestions?.length) {
            setSuggestions(res.suggestions);
          }
        }
        setCursor(res.next_cursor);
        setHasMore(res.has_more);
      } catch {
        const msg = "Failed to load feed";
        if (!nextCursor) {
          setError(msg);
        }
        addToast(msg, "error");
      } finally {
        setLoading(false);
      }
    },
    [addToast],
  );

  useEffect(() => {
    loadFeed();
  }, [loadFeed]);

  // Clear active room on unmount
  useEffect(() => {
    return () => setActiveRoom(null);
  }, [setActiveRoom]);

  const loadMore = useCallback(async () => {
    if (!loadingMoreRef.current && hasMore && cursor) {
      loadingMoreRef.current = true;
      setLoadingMore(true);
      try {
        await loadFeed(cursor);
      } finally {
        setLoadingMore(false);
        loadingMoreRef.current = false;
      }
    }
  }, [cursor, hasMore, loadFeed]);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          loadMore();
        }
      },
      { rootMargin: "800px 0px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [loadMore]);

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    );
  }

  if (error && items.length === 0) {
    return (
      <div className="text-center py-12">
        <p className="text-[var(--color-error)]">{error}</p>
        <button
          onClick={() => {
            setLoading(true);
            loadFeed();
          }}
          className="mt-4 px-4 py-2 rounded-lg bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] border border-[var(--color-border)] text-sm font-medium transition-colors"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="px-4 pt-4 sm:px-0 sm:pt-0">
        <TrendingTags />
      </div>
      {items.length === 0 ? (
        <div className="px-4 text-center py-12 text-[var(--color-text-muted)]">
          <p className="text-lg">No content yet</p>
          <p className="text-sm mt-2">
            Follow people or post something to see it here
          </p>
        </div>
      ) : (
        <>
          {items.map((item, index) => (
            <Fragment key={item.id}>
              <div ref={observe} data-content-id={item.id}>
                <FeedCard
                  content={item}
                  onClick={() => setSelected(item)}
                />
              </div>
              {index === 4 && suggestions.length > 0 && (
                <InlineSuggestionGroup suggestions={suggestions} />
              )}
            </Fragment>
          ))}
          <div ref={sentinelRef} className="h-8" />
          {loadingMore && (
            <div className="flex justify-center py-4">
              <Spinner size="sm" />
            </div>
          )}
        </>
      )}
      {selected && (
        <ContentModal content={selected} onClose={() => setSelected(null)} />
      )}
    </div>
  );
}
