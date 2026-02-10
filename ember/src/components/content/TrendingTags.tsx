import { useEffect, useState, useCallback } from "react";
import type { Tag } from "@pulse/drift/types";
import { getTags } from "../../api/tags";

interface Props {
  limit?: number;
}

export default function TrendingTags({ limit = 10 }: Props) {
  const [tags, setTags] = useState<Tag[]>([]);
  const [error, setError] = useState(false);
  const [retryCount, setRetryCount] = useState(0);

  useEffect(() => {
    let cancelled = false;

    getTags()
      .then((allTags) => {
        if (!cancelled) {
          setTags(allTags.slice(0, limit));
          setError(false);
        }
      })
      .catch(() => {
        if (!cancelled) setError(true);
      });

    return () => {
      cancelled = true;
    };
  }, [limit, retryCount]);

  const handleRetry = useCallback(() => {
    setError(false);
    setRetryCount((c) => c + 1);
  }, []);

  if (error) {
    return (
      <section className="bg-[var(--color-surface)] rounded-lg p-4 border border-[var(--color-border)]">
        <div className="flex items-center justify-between">
          <p className="text-xs text-red-400">Could not load trending tags</p>
          <button
            onClick={handleRetry}
            className="text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
          >
            Retry
          </button>
        </div>
      </section>
    );
  }

  if (tags.length === 0) {
    return null;
  }

  return (
    <section className="bg-[var(--color-surface)] rounded-lg p-4 border border-[var(--color-border)]">
      <h3 className="text-sm font-semibold mb-2">Trending tags</h3>
      <div className="flex flex-wrap gap-1.5">
        {tags.map((tag) => (
          <span
            key={tag.id}
            className="text-xs px-2 py-0.5 rounded-full bg-[var(--color-surface-hover)] text-[var(--color-text-muted)]"
          >
            #{tag.name}
          </span>
        ))}
      </div>
    </section>
  );
}
