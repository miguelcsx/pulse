import { useEffect } from "react";
import Modal from "../ui/Modal";
import type { Content } from "@pulse/drift/types";
import ReactionBar from "./ReactionBar";
import MediaFallback from "../ui/MediaFallback";
import { trackView } from "../../api/events";

interface Props {
  content: Content;
  onClose: () => void;
}

function renderContent(content: Content) {
  switch (content.content_type) {
    case "image":
      return (
        <MediaFallback
          src={content.media_url}
          alt={content.body || "Image post"}
          type="image"
          className="w-full max-h-[70vh] object-contain rounded"
        />
      );
    case "video":
    case "short_video":
      return (
        <MediaFallback
          src={content.media_url}
          alt={content.body || "Video post"}
          type={content.content_type}
          className="w-full max-h-[70vh] rounded bg-black"
          controls
          playsInline
        />
      );
    case "text":
      return (
        <div className="rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
          <p className="whitespace-pre-wrap text-sm leading-relaxed">
            {content.body}
          </p>
        </div>
      );
    default:
      return null;
  }
}

export default function ContentModal({ content, onClose }: Props) {
  useEffect(() => {
    trackView("content", content.id);
  }, [content.id]);

  return (
    <Modal open onClose={onClose} title="">
      {renderContent(content)}
      {content.content_type !== "text" && content.body && (
        <p className="mt-3 text-sm whitespace-pre-wrap">{content.body}</p>
      )}
      {content.tags && content.tags.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1.5">
          {content.tags.map((tag) => (
            <span
              key={tag.id}
              className="text-xs px-2 py-0.5 rounded-full bg-[var(--color-surface-hover)] text-[var(--color-text-muted)]"
            >
              #{tag.name}
            </span>
          ))}
        </div>
      )}
      <ReactionBar contentId={content.id} initialCounts={content.reactions} />
      <p className="mt-2 text-xs text-[var(--color-text-muted)]">
        by {content.creator?.display_name}
      </p>
    </Modal>
  );
}
