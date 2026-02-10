import { useState, useEffect, useCallback } from "react";
import { getSuggestions } from "../api/social";
import SuggestionCard from "../components/social/SuggestionCard";
import Spinner from "../components/ui/Spinner";
import { useUiStore } from "../store/uiStore";
import { usePageTitle } from "../hooks/usePageTitle";
import type { Suggestion } from "@pulse/drift/types";

export default function Suggestions() {
  usePageTitle("Discover");
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [retryCount, setRetryCount] = useState(0);
  const addToast = useUiStore((s) => s.addToast);

  useEffect(() => {
    let cancelled = false;

    getSuggestions()
      .then((data) => {
        if (!cancelled) {
          setSuggestions(data);
          setLoading(false);
          setError(null);
        }
      })
      .catch(() => {
        if (!cancelled) {
          const msg = "Failed to load suggestions";
          setError(msg);
          setLoading(false);
          addToast(msg, "error");
        }
      });

    return () => {
      cancelled = true;
    };
  }, [addToast, retryCount]);

  const handleRetry = useCallback(() => {
    setLoading(true);
    setError(null);
    setRetryCount((c) => c + 1);
  }, []);

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
        <p className="text-red-400">{error}</p>
        <button
          onClick={handleRetry}
          className="mt-4 px-4 py-2 rounded-lg bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] border border-[var(--color-border)] text-sm font-medium transition-colors"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">People You Might Like</h2>
      {suggestions.length === 0 ? (
        <p className="text-[var(--color-text-muted)] text-sm">
          Post content with hashtags to get personalized suggestions
        </p>
      ) : (
        suggestions.map((s) => (
          <SuggestionCard key={s.user.id} suggestion={s} />
        ))
      )}
    </div>
  );
}
