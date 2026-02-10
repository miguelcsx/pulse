import { useState, useEffect, useCallback } from "react";
import { useParams } from "react-router-dom";
import { getUserProfile } from "../api/social";
import { getUserContent } from "../api/content";
import { trackView } from "../api/events";
import { useAuthStore } from "../store/authStore";
import { useUiStore } from "../store/uiStore";
import { usePageTitle } from "../hooks/usePageTitle";
import FollowButton from "../components/social/FollowButton";
import Spinner from "../components/ui/Spinner";
import ContentModal from "../components/feed/ContentModal";
import MediaFallback from "../components/ui/MediaFallback";
import type { Content, UserProfile } from "@pulse/drift/types";

function Avatar({
  src,
  fallbackChar,
  size = "md",
}: {
  src?: string;
  fallbackChar: string;
  size?: "md" | "lg";
}) {
  const sizeClasses = size === "lg" ? "w-16 h-16 text-xl" : "w-10 h-10 text-sm";

  if (src) {
    return (
      <img
        src={src}
        alt=""
        className={`${sizeClasses} rounded-full object-cover bg-indigo-600`}
        onError={(e) => {
          // Fall back to initial on broken avatar URL
          const target = e.currentTarget;
          target.style.display = "none";
          target.nextElementSibling?.classList.remove("hidden");
        }}
      />
    );
  }

  return (
    <div
      className={`${sizeClasses} rounded-full bg-indigo-600 flex items-center justify-center font-medium`}
    >
      {fallbackChar}
    </div>
  );
}

function ProfileContentPreview({
  content,
  onClick,
}: {
  content: Content;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-lg overflow-hidden border border-[var(--color-border)] bg-[var(--color-surface)] text-left"
    >
      {content.content_type === "image" && (
        <MediaFallback
          src={content.media_url}
          alt={content.body || "Image content"}
          type="image"
          className="w-full aspect-square object-cover"
          loading="lazy"
        />
      )}
      {(content.content_type === "video" ||
        content.content_type === "short_video") && (
        <MediaFallback
          src={content.media_url}
          alt={content.body || "Video content"}
          type={content.content_type}
          className="w-full aspect-square object-cover bg-black"
          muted
          playsInline
          preload="metadata"
        />
      )}
      {content.content_type === "text" && (
        <div className="aspect-square p-3 bg-[var(--color-surface-hover)]">
          <p className="text-sm line-clamp-8 whitespace-pre-wrap">
            {content.body}
          </p>
        </div>
      )}
    </button>
  );
}

export default function Profile() {
  const { id } = useParams<{ id: string }>();
  const currentUser = useAuthStore((s) => s.user);
  const [profile, setProfile] = useState<UserProfile | null>(null);
  usePageTitle(
    profile?.display_name ? `${profile.display_name}'s Profile` : "Profile",
  );
  const [content, setContent] = useState<Content[]>([]);
  const [selected, setSelected] = useState<Content | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [retryCount, setRetryCount] = useState(0);
  const addToast = useUiStore((s) => s.addToast);

  useEffect(() => {
    if (!id) return;

    let cancelled = false;

    Promise.all([getUserProfile(id), getUserContent(id, undefined, 30)])
      .then(([profileData, contentData]) => {
        if (!cancelled) {
          setProfile(profileData);
          setContent(contentData.items);
          setLoading(false);
          setError(null);
          trackView("user", id);
        }
      })
      .catch(() => {
        if (!cancelled) {
          const msg = "Failed to load profile";
          setError(msg);
          setLoading(false);
          addToast(msg, "error");
        }
      });

    return () => {
      cancelled = true;
    };
  }, [id, addToast, retryCount]);

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

  if (!profile) {
    return <p className="text-[var(--color-text-muted)]">User not found</p>;
  }

  const isMe = currentUser?.id === profile.id;

  return (
    <div className="space-y-6">
      <div className="bg-[var(--color-surface)] rounded-lg p-6 border border-[var(--color-border)]">
        <div className="flex items-center gap-4">
          <Avatar
            src={profile.avatar_url}
            fallbackChar={profile.display_name?.[0] || "?"}
            size="lg"
          />
          <div className="flex-1">
            <h2 className="text-lg font-semibold">{profile.display_name}</h2>
            <p className="text-sm text-[var(--color-text-muted)]">
              @{profile.handle}
            </p>
          </div>
          {!isMe && (
            <FollowButton
              userId={profile.id}
              initialFollowing={profile.is_following}
            />
          )}
        </div>

        {profile.bio && <p className="text-sm mt-4">{profile.bio}</p>}
        {profile.location && (
          <p className="text-xs text-[var(--color-text-muted)] mt-1">
            {profile.location}
          </p>
        )}

        <div className="flex gap-6 mt-4 text-sm">
          <div>
            <span className="font-medium">{profile.content_count}</span>
            <span className="text-[var(--color-text-muted)] ml-1">posts</span>
          </div>
          <div>
            <span className="font-medium">{profile.follower_count}</span>
            <span className="text-[var(--color-text-muted)] ml-1">
              followers
            </span>
          </div>
          <div>
            <span className="font-medium">{profile.following_count}</span>
            <span className="text-[var(--color-text-muted)] ml-1">
              following
            </span>
          </div>
        </div>
      </div>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">Recent posts</h3>
        {content.length === 0 ? (
          <p className="text-sm text-[var(--color-text-muted)]">
            No content yet.
          </p>
        ) : (
          <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
            {content.map((item) => (
              <ProfileContentPreview
                key={item.id}
                content={item}
                onClick={() => setSelected(item)}
              />
            ))}
          </div>
        )}
      </section>

      {selected && (
        <ContentModal content={selected} onClose={() => setSelected(null)} />
      )}
    </div>
  );
}
