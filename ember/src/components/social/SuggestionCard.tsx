import { Link } from "react-router-dom";
import FollowButton from "./FollowButton";
import type { Suggestion } from "@pulse/drift/types";

interface Props {
  suggestion: Suggestion;
}

export default function SuggestionCard({ suggestion }: Props) {
  const { user, shared_tags, common_tags, bridge } = suggestion;

  return (
    <div className="bg-[var(--color-surface)] rounded-lg p-4 border border-[var(--color-border)]">
      <div className="flex items-center justify-between">
        <Link to={`/profile/${user.id}`} className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-indigo-600 flex items-center justify-center text-sm font-medium">
            {user.display_name?.[0] || "?"}
          </div>
          <div>
            <p className="font-medium text-sm">{user.display_name}</p>
            <p className="text-xs text-[var(--color-text-muted)]">@{user.handle}</p>
          </div>
        </Link>
        <FollowButton userId={user.id} />
      </div>
      <div className="mt-3">
        <p className="text-sm mb-2">{bridge}</p>
        <p className="text-xs text-[var(--color-text-muted)] mb-1.5">
          {shared_tags} shared {shared_tags === 1 ? "tag" : "tags"}
        </p>
        <div className="flex flex-wrap gap-1.5">
          {common_tags?.map((tag) => (
            <span
              key={tag.id}
              className="text-xs px-2 py-0.5 rounded-full bg-indigo-900/30 text-indigo-300"
            >
              {tag.name}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}
