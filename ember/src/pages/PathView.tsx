import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { getPath, followPath, unfollowPath } from "../api/paths";
import { trackPathFollow, trackView } from "../api/events";
import { useUiStore } from "../store/uiStore";
import { useAuthStore } from "../store/authStore";
import { usePageTitle } from "../hooks/usePageTitle";
import Button from "../components/ui/Button";
import Spinner from "../components/ui/Spinner";
import MediaFallback from "../components/ui/MediaFallback";
import { useVideoAutoplay } from "../hooks/useVideoAutoplay";
import type { Content, Path } from "@pulse/drift/types";

function PathVideoPreview({
  content,
  className,
}: {
  content: Content;
  className: string;
}) {
  const videoRef = useVideoAutoplay({ threshold: 0.5 });

  return (
    <MediaFallback
      src={content.media_url}
      alt={content.body || "Video content"}
      type={content.content_type as "video" | "short_video"}
      className={className}
      preload="metadata"
      muted
      playsInline
      videoRef={videoRef}
    />
  );
}

function renderContentPreview(content: Content, className: string) {
  switch (content.content_type) {
    case "image":
      return (
        <MediaFallback
          src={content.media_url}
          alt={content.body || "Image content"}
          type="image"
          className={className}
          loading="lazy"
        />
      );
    case "video":
    case "short_video":
      return <PathVideoPreview content={content} className={className} />;
    case "text":
      return (
        <div className="p-4 bg-[var(--color-surface)] rounded-[var(--radius-sm)]">
          <p className="text-sm line-clamp-5 whitespace-pre-wrap text-[var(--color-text-secondary)]">
            {content.body}
          </p>
        </div>
      );
    default:
      return null;
  }
}

export default function PathView() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const addToast = useUiStore((s) => s.addToast);
  const user = useAuthStore((s) => s.user);

  const [path, setPath] = useState<Path | null>(null);
  const [loading, setLoading] = useState(true);
  usePageTitle(path?.title ? path.title : "Path");

  useEffect(() => {
    if (!id) return;
    getPath(id)
      .then((data) => {
        setPath(data);
        trackView("path", id);
      })
      .catch(() => addToast("Path not found", "error"))
      .finally(() => setLoading(false));
  }, [id, addToast]);

  const handleFollow = async () => {
    if (!id) return;
    try {
      await followPath(id);
      trackPathFollow(id);
      addToast("Following path", "success");
      if (path) {
        setPath({
          ...path,
          follower_count: path.follower_count + 1,
          is_following: true,
        });
      }
    } catch {
      addToast("Failed to follow", "error");
    }
  };

  const handleUnfollow = async () => {
    if (!id) return;
    try {
      await unfollowPath(id);
      addToast("Unfollowed path", "success");
      if (path) {
        setPath({
          ...path,
          follower_count: Math.max(0, path.follower_count - 1),
          is_following: false,
        });
      }
    } catch {
      addToast("Failed to unfollow", "error");
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center py-16">
        <Spinner size="lg" />
      </div>
    );
  }

  if (!path) {
    return (
      <p className="text-[var(--color-text-muted)] text-center py-16">
        Path not found
      </p>
    );
  }

  const isOwner = user?.id === path.creator_id;

  return (
    <div className="space-y-6 pb-4">
      <button
        onClick={() => navigate(-1)}
        className="flex items-center gap-1 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors pt-2"
      >
        <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <polyline points="15 18 9 12 15 6" />
        </svg>
        Back
      </button>

      <section>
        <h2 className="text-xl font-semibold">{path.title}</h2>
        {path.description && (
          <p className="text-sm text-[var(--color-text-muted)] mt-1">
            {path.description}
          </p>
        )}
        <div className="flex items-center gap-4 mt-2 text-xs text-[var(--color-text-muted)]">
          <span>by {path.creator?.display_name}</span>
          <span className="tabular-nums">{path.follower_count} followers</span>
        </div>
        {!isOwner && (
          <div className="mt-3">
            {path.is_following ? (
              <Button size="sm" variant="secondary" onClick={handleUnfollow}>
                Unfollow
              </Button>
            ) : (
              <Button size="sm" variant="accent" onClick={handleFollow}>
                Follow path
              </Button>
            )}
          </div>
        )}
      </section>

      <section className="space-y-3">
        {path.items?.length === 0 ? (
          <p className="text-[var(--color-text-muted)] text-sm text-center py-8">
            No items in this path yet.
          </p>
        ) : (
          path.items?.map((item, i) => (
            <div
              key={item.id}
              className="rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border)] overflow-hidden"
            >
              <div className="flex items-center gap-3 p-3">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-[var(--color-surface)] text-xs font-medium text-[var(--color-text-muted)] tabular-nums">
                  {i + 1}
                </span>
                {item.note && (
                  <p className="text-sm text-[var(--color-text-secondary)]">
                    {item.note}
                  </p>
                )}
              </div>
              {item.content &&
                renderContentPreview(
                  item.content,
                  "w-full aspect-video object-cover",
                )}
            </div>
          ))
        )}
      </section>
    </div>
  );
}
