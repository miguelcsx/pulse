import { Link } from "react-router-dom";
import type { FeedItem } from "@pulse/drift/types";
import ReactionBar from "./ReactionBar";
import MediaFallback from "../ui/MediaFallback";
import { useDwell } from "../../hooks/useDwell";
import { trackView } from "../../api/events";
import { useVideoAutoplay } from "../../hooks/useVideoAutoplay";

interface Props {
  content: FeedItem;
  onClick?: () => void;
}

function FeedVideoPreview({ content }: { content: FeedItem }) {
  const videoRef = useVideoAutoplay();

  return (
    <MediaFallback
      src={content.media_url}
      alt={content.body || "Video post"}
      type={content.content_type as "video" | "short_video"}
      className="mx-auto w-full max-h-[70dvh] aspect-[9/16] object-cover bg-black"
      playsInline
      muted
      preload="metadata"
      videoRef={videoRef}
    />
  );
}

function renderPreview(content: FeedItem) {
  switch (content.content_type) {
    case "image":
      return (
        <MediaFallback
          src={content.media_url}
          alt={content.body || "Image post"}
          type="image"
          className="w-full aspect-[4/5] object-cover"
          loading="lazy"
        />
      );
    case "video":
    case "short_video":
      return <FeedVideoPreview content={content} />;
    case "text":
      return (
        <div className="w-full min-h-[45dvh] p-4 text-left bg-gradient-to-br from-[var(--color-primary-subtle)] to-[var(--color-surface)]">
          <p className="text-sm whitespace-pre-wrap line-clamp-6">
            {content.body}
          </p>
        </div>
      );
    default:
      return null;
  }
}

export default function FeedCard({ content, onClick }: Props) {
  const dwellRef = useDwell("content", content.id);

  const handleClick = () => {
    trackView("content", content.id);
    onClick?.();
  };

  return (
    <article
      ref={dwellRef}
      data-content-id={content.id}
      className="snap-start bg-[var(--color-surface)] rounded-none sm:rounded-lg overflow-hidden border-y sm:border border-[var(--color-border)] min-h-[calc(100dvh-9.5rem)] sm:min-h-0"
    >
      <div className="flex items-center gap-3 p-3">
        <div className="w-8 h-8 rounded-full bg-[var(--color-primary)] flex items-center justify-center text-sm font-medium text-white">
          {content.creator?.display_name?.[0] || "?"}
        </div>
        <Link
          to={`/profile/${content.creator_id}`}
          className="text-sm font-medium hover:text-[var(--color-primary)]"
        >
          {content.creator?.display_name || "Unknown"}
        </Link>
        {content.room_context && (
          <span className="ml-auto flex items-center gap-1.5 text-xs text-[var(--color-text-muted)]">
            <span className="inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500" />
            {content.room_context.member_count}
          </span>
        )}
      </div>

      <button
        type="button"
        onClick={handleClick}
        className="w-full cursor-pointer text-left"
      >
        {renderPreview(content)}
      </button>

      {content.content_type !== "text" && content.body && (
        <div className="p-3 pb-0">
          <p className="text-sm whitespace-pre-wrap">{content.body}</p>
        </div>
      )}

      {content.tags && content.tags.length > 0 && (
        <div className="px-3 pb-0 pt-3 flex flex-wrap gap-1.5">
          {content.tags.map((tag) => (
            <span
              key={tag.id}
              className="text-xs px-2 py-0.5 rounded-full bg-[var(--color-tag-bg)] text-[var(--color-tag-text)]"
            >
              #{tag.name}
            </span>
          ))}
        </div>
      )}

      <div className="px-3 pb-3">
        <ReactionBar contentId={content.id} initialCounts={content.reactions} />
      </div>
    </article>
  );
}
