import { Link } from "react-router-dom";
import FollowButton from "./FollowButton";
import type { Suggestion, SuggestionType } from "@pulse/drift/types";

const TYPE_STYLES: Record<SuggestionType, { border: string; label: string }> = {
  closest_twin: { border: "var(--color-primary)", label: "SIMILAR VIBES" },
  adjacent_taste: { border: "var(--color-tag-text)", label: "SHARED TASTE" },
  path_affinity: { border: "#8b5cf6", label: "PATH CONNECTION" },
  serendipity: { border: "#f59e0b", label: "FRESH PERSPECTIVE" },
};

interface Props {
  suggestion: Suggestion;
  compact?: boolean;
}

export default function SuggestionCard({ suggestion, compact }: Props) {
  const { user, shared_tags, common_tags, bridge, suggestion_type } = suggestion;
  const style = suggestion_type ? TYPE_STYLES[suggestion_type] : undefined;
  const borderColor = style?.border ?? "var(--color-primary)";

  return (
    <div
      className={`glass premium-card rounded-xl ${compact ? "p-3" : "p-5"}`}
      style={{ borderLeftWidth: 3, borderLeftColor: borderColor }}
    >
      <div className="flex items-center justify-between">
        <Link to={`/profile/${user.id}`} className="flex items-center gap-3 min-w-0">
          <div className={`${compact ? "w-8 h-8 text-xs" : "w-10 h-10 text-sm"} rounded-full bg-[var(--color-primary)] flex items-center justify-center font-medium text-white shrink-0`}>
            {user.display_name?.[0] || "?"}
          </div>
          <div className="min-w-0">
            <p className={`font-medium ${compact ? "text-xs" : "text-sm"} truncate`}>{user.display_name}</p>
            <p className={`${compact ? "text-[10px]" : "text-xs"} text-[var(--color-text-muted)] truncate`}>@{user.handle}</p>
          </div>
        </Link>
        <FollowButton userId={user.id} />
      </div>
      <div className={compact ? "mt-2" : "mt-3"}>
        {style && (
          <p
            className={`${compact ? "text-[9px] mb-1" : "text-[10px] mb-1.5"} font-semibold tracking-wider uppercase`}
            style={{ color: borderColor }}
          >
            {style.label}
          </p>
        )}
        <p
          className={`${compact ? "text-xs" : "text-sm"} ${compact ? "mb-1" : "mb-2"} border-l-2 pl-3`}
          style={{ borderColor }}
        >
          {bridge}
        </p>
        {!compact && (
          <>
            <p className="text-xs text-[var(--color-text-muted)] mb-1.5">
              {shared_tags} shared {shared_tags === 1 ? "tag" : "tags"}
            </p>
            <div className="flex flex-wrap gap-1.5">
              {common_tags?.map((tag) => (
                <span
                  key={tag.id}
                  className="text-xs px-2 py-0.5 rounded-full bg-[var(--color-tag-bg)] text-[var(--color-tag-text)]"
                >
                  {tag.name}
                </span>
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
