import SuggestionCard from "../social/SuggestionCard";
import type { Suggestion } from "@pulse/drift/types";

interface Props {
  suggestions: Suggestion[];
}

export default function InlineSuggestionGroup({ suggestions }: Props) {
  if (suggestions.length === 0) return null;

  return (
    <div className="border-y border-[var(--color-border)] sm:rounded-lg sm:border sm:mx-0 py-4 px-4 bg-[var(--color-bg)]">
      <p className="text-[10px] font-semibold tracking-wider uppercase text-[var(--color-text-muted)] mb-3">
        People you might connect with
      </p>
      <div className="space-y-2">
        {suggestions.map((s) => (
          <SuggestionCard key={s.user.id} suggestion={s} compact />
        ))}
      </div>
    </div>
  );
}
