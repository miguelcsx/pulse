import { useEffect, useState } from "react";
import Modal from "../ui/Modal";
import type { Content } from "@pulse/drift/types";
import ReactionBar from "./ReactionBar";
import MediaFallback from "../ui/MediaFallback";
import { trackView } from "../../api/events";
import { deleteContent } from "../../api/content";
import { useAuthStore } from "../../store/authStore";
import { useUiStore } from "../../store/uiStore";

interface Props {
  content: Content;
  onClose: () => void;
  onDelete?: () => void;
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
          autoPlay
          muted
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

export default function ContentModal({ content, onClose, onDelete }: Props) {
  const currentUser = useAuthStore((s) => s.user);
  const addToast = useUiStore((s) => s.addToast);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    trackView("content", content.id);
  }, [content.id]);

  const isOwner = currentUser?.id === content.creator_id;

  const handleDelete = async () => {
    if (!confirm("Delete this content? This cannot be undone.")) return;
    setDeleting(true);
    try {
      await deleteContent(content.id);
      addToast("Content deleted", "success");
      onDelete?.();
      onClose();
    } catch {
      addToast("Failed to delete content", "error");
    } finally {
      setDeleting(false);
    }
  };

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
              className="text-xs px-2 py-0.5 rounded-full bg-[var(--color-tag-bg)] text-[var(--color-tag-text)]"
            >
              #{tag.name}
            </span>
          ))}
        </div>
      )}
      <ReactionBar contentId={content.id} initialCounts={content.reactions} />
      <div className="mt-2 flex items-center justify-between">
        <p className="text-xs text-[var(--color-text-muted)]">
          by {content.creator?.display_name}
        </p>
        {isOwner && (
          <button
            onClick={handleDelete}
            disabled={deleting}
            className="text-xs px-3 py-1 rounded bg-[var(--color-error)] hover:opacity-90 text-white disabled:opacity-50 transition-colors"
          >
            {deleting ? "Deleting..." : "Delete"}
          </button>
        )}
      </div>
    </Modal>
  );
}
