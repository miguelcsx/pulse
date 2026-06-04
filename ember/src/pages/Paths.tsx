import { useState, useEffect, useCallback } from "react";
import { Link } from "react-router-dom";
import { getPaths } from "../api/paths";
import Spinner from "../components/ui/Spinner";
import Button from "../components/ui/Button";
import { useUiStore } from "../store/uiStore";
import { usePageTitle } from "../hooks/usePageTitle";
import type { Path } from "@pulse/drift/types";


export default function Paths() {
  usePageTitle("Paths");
  const [paths, setPaths] = useState<Path[]>([]);
  const [cursor, setCursor] = useState("");
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const addToast = useUiStore((s) => s.addToast);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    getPaths()
      .then((data) => {
        setPaths(data.items);
        setCursor(data.next_cursor);
        setHasMore(data.has_more);
      })
      .catch(() => {
        const msg = "Failed to load paths";
        setError(msg);
        addToast(msg, "error");
      })
      .finally(() => setLoading(false));
  }, [addToast]);

  useEffect(() => {
    load();
  }, [load]);

  const loadMore = async () => {
    if (!cursor || !hasMore || loadingMore) return;

    setLoadingMore(true);
    try {
      const data = await getPaths(cursor);
      setPaths((prev) => [...prev, ...data.items]);
      setCursor(data.next_cursor);
      setHasMore(data.has_more);
    } catch {
      addToast("Failed to load more paths", "error");
    } finally {
      setLoadingMore(false);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-12">
        <p className="text-[var(--color-error)]">{error}</p>
        <button
          onClick={load}
          className="mt-4 px-4 py-2 rounded-lg bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] border border-[var(--color-border)] text-sm font-medium transition-colors"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">Relationship trails</h2>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Curated sequences that keep old moments connected to people and
          context.
        </p>
      </div>
      {paths.length === 0 ? (
        <p className="text-[var(--color-text-muted)] text-sm">
          No curated paths yet. Pulse still builds a live affinity path from
          moments, asks, and responses.
        </p>
      ) : (
        <>
          {paths.map((path) => (
            <Link
              key={path.id}
              to={`/paths/${path.id}`}
              className="block bg-[var(--color-surface)] rounded-lg p-4 border border-[var(--color-border)] hover:border-[var(--color-border-emphasis)] transition-colors"
            >
              <h3 className="font-medium">{path.title}</h3>
              {path.description && (
                <p className="text-sm text-[var(--color-text-muted)] mt-1">
                  {path.description}
                </p>
              )}
              <div className="flex items-center gap-4 mt-2 text-xs text-[var(--color-text-muted)]">
                <span>{path.items?.length || 0} items</span>
                <span>{path.follower_count} people on this path</span>
                <span>by {path.creator?.display_name}</span>
              </div>
            </Link>
          ))}
          {hasMore && (
            <div className="flex justify-center pt-2">
              <Button
                variant="secondary"
                size="sm"
                onClick={loadMore}
                loading={loadingMore}
              >
                Load More Paths
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
