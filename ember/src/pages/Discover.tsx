import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { getDiscover } from "../api/discover";
import SuggestionCard from "../components/social/SuggestionCard";
import Spinner from "../components/ui/Spinner";
import { useUiStore } from "../store/uiStore";
import { usePageTitle } from "../hooks/usePageTitle";
import type { DiscoverResponse, Suggestion } from "@pulse/drift/types";

function SuggestionSection({
  title,
  subtitle,
  suggestions,
}: {
  title: string;
  subtitle: string;
  suggestions: Suggestion[];
}) {
  if (suggestions.length === 0) return null;
  return (
    <section>
      <h2 className="text-lg font-semibold">{title}</h2>
      <p className="text-xs text-[var(--color-text-muted)] mb-4">{subtitle}</p>
      <div className="space-y-3">
        {suggestions.map((s) => (
          <SuggestionCard key={s.user.id} suggestion={s} />
        ))}
      </div>
    </section>
  );
}

export default function Discover() {
  usePageTitle("Discover");
  const [data, setData] = useState<DiscoverResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const addToast = useUiStore((s) => s.addToast);

  useEffect(() => {
    let cancelled = false;
    getDiscover()
      .then((res) => {
        if (!cancelled) setData(res);
      })
      .catch(() => {
        if (!cancelled) {
          setError("Failed to load discover");
          addToast("Failed to load discover", "error");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [addToast]);

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="text-center py-12">
        <p className="text-[var(--color-error)]">{error}</p>
      </div>
    );
  }

  const hasBuckets =
    (data.closest_twins?.length ?? 0) > 0 ||
    (data.adjacent_taste?.length ?? 0) > 0 ||
    (data.serendipity?.length ?? 0) > 0;

  // Check for path_affinity suggestions in the flat list
  const pathSuggestions = data.suggestions.filter(
    (s) => s.suggestion_type === "path_affinity",
  );
  const nonPathSuggestions = data.suggestions.filter(
    (s) => s.suggestion_type !== "path_affinity",
  );

  return (
    <div className="space-y-8">
      {/* Path Connections */}
      {pathSuggestions.length > 0 && (
        <SuggestionSection
          title="Path Connections"
          subtitle="People whose paths resonate with you"
          suggestions={pathSuggestions}
        />
      )}

      {/* Bucketed sections when available */}
      {hasBuckets ? (
        <>
          <SuggestionSection
            title="Similar Vibes"
            subtitle="People who experience content like you do"
            suggestions={data.closest_twins ?? []}
          />
          <SuggestionSection
            title="Shared Taste"
            subtitle="People posting about similar things"
            suggestions={data.adjacent_taste ?? []}
          />
          <SuggestionSection
            title="Fresh Perspectives"
            subtitle="Different style, one strong connection"
            suggestions={data.serendipity ?? []}
          />
        </>
      ) : (
        /* Fallback: flat suggestions (backward compat) */
        nonPathSuggestions.length > 0 && (
          <section>
            <h2 className="text-lg font-semibold mb-4">People You Might Like</h2>
            <div className="space-y-3">
              {nonPathSuggestions.map((s) => (
                <SuggestionCard key={s.user.id} suggestion={s} />
              ))}
            </div>
          </section>
        )
      )}

      {!hasBuckets && nonPathSuggestions.length === 0 && pathSuggestions.length === 0 && (
        <section>
          <h2 className="text-lg font-semibold mb-4">People You Might Like</h2>
          <p className="text-sm text-[var(--color-text-muted)]">
            Post content with hashtags to get personalized suggestions
          </p>
        </section>
      )}

      {/* Active Vibes */}
      <section>
        <h2 className="text-lg font-semibold mb-4">Active Vibes</h2>
        {data.rooms.length === 0 ? (
          <p className="text-sm text-[var(--color-text-muted)]">
            No active rooms right now
          </p>
        ) : (
          <div className="grid grid-cols-2 gap-3">
            {data.rooms.map((room) => (
              <Link
                key={room.id}
                to={`/rooms/${room.id}`}
                className="bg-[var(--color-surface)] rounded-lg p-4 border border-[var(--color-border)] hover:border-[var(--color-primary)] transition-colors"
              >
                <div className="flex items-center gap-2 mb-2">
                  <span className="inline-flex h-2 w-2 rounded-full bg-emerald-500" />
                  <span className="text-xs text-[var(--color-text-muted)]">
                    {room.member_count} {room.member_count === 1 ? "person" : "people"}
                  </span>
                </div>
                <div className="flex flex-wrap gap-1">
                  {room.tags?.map((tag) => (
                    <span
                      key={tag.id}
                      className="text-xs px-2 py-0.5 rounded-full bg-[var(--color-tag-bg)] text-[var(--color-tag-text)]"
                    >
                      #{tag.name}
                    </span>
                  ))}
                </div>
              </Link>
            ))}
          </div>
        )}
      </section>

      {/* Paths to Explore */}
      <section>
        <h2 className="text-lg font-semibold mb-4">Paths to Explore</h2>
        {data.paths.length === 0 ? (
          <p className="text-sm text-[var(--color-text-muted)]">
            No paths available yet
          </p>
        ) : (
          <div className="space-y-3">
            {data.paths.map((path) => (
              <Link
                key={path.id}
                to={`/paths/${path.id}`}
                className="block bg-[var(--color-surface)] rounded-lg p-4 border border-[var(--color-border)] hover:border-[var(--color-primary)] transition-colors"
              >
                <h3 className="font-medium text-sm">{path.title}</h3>
                {path.description && (
                  <p className="text-xs text-[var(--color-text-muted)] mt-1 line-clamp-2">
                    {path.description}
                  </p>
                )}
                <div className="flex items-center gap-3 mt-2 text-xs text-[var(--color-text-muted)]">
                  <span>{path.items?.length || 0} items</span>
                  <span>{path.follower_count} followers</span>
                  {path.creator && <span>by {path.creator.display_name}</span>}
                </div>
              </Link>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
