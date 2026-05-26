import { useState, useEffect, useCallback } from "react";
import { useParams, Link } from "react-router-dom";
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
  const sizeClasses =
    size === "lg" ? "w-20 h-20 text-2xl" : "w-10 h-10 text-sm";

  if (src) {
    return (
      <img
        src={src}
        alt=""
        className={`${sizeClasses} rounded-full object-cover bg-[var(--color-surface-active)]`}
        onError={(e) => {
          const target = e.currentTarget;
          target.style.display = "none";
          target.nextElementSibling?.classList.remove("hidden");
        }}
      />
    );
  }

  return (
    <div
      className={`${sizeClasses} rounded-full bg-[var(--color-surface-active)] flex items-center justify-center font-medium text-[var(--color-text-secondary)]`}
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
      className="rounded-[var(--radius-sm)] overflow-hidden bg-[var(--color-surface)] text-left hover:opacity-90 transition-opacity"
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
        <div className="aspect-square p-3 bg-[var(--color-surface)] flex items-center">
          <p className="text-xs line-clamp-6 whitespace-pre-wrap text-[var(--color-text-secondary)]">
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
    profile?.display_name ? `${profile.display_name}` : "Profile",
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
      <div className="flex justify-center py-16">
        <Spinner size="lg" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-16">
        <p className="text-[var(--color-error)]">{error}</p>
        <button
          onClick={handleRetry}
          className="mt-4 px-4 py-2.5 rounded-[var(--radius-sm)] bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-sm font-medium transition-colors"
        >
          Retry
        </button>
      </div>
    );
  }

  if (!profile) {
    return (
      <p className="py-16 text-center text-[var(--color-text-muted)]">
        User not found
      </p>
    );
  }

  const isMe = currentUser?.id === profile.id;

  return (
    <div className="space-y-6 pb-4">
      {/* Profile header */}
      <section className="pt-4 text-center">
        <div className="flex justify-center mb-3">
          <Avatar
            src={profile.avatar_url}
            fallbackChar={profile.display_name?.[0] || "?"}
            size="lg"
          />
        </div>
        <h2 className="text-xl font-semibold">{profile.display_name}</h2>
        <p className="text-sm text-[var(--color-text-muted)]">
          @{profile.handle}
        </p>
        {profile.bio && (
          <p className="mt-2 text-sm text-[var(--color-text-secondary)] max-w-xs mx-auto">
            {profile.bio}
          </p>
        )}
        {profile.location && (
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">
            {profile.location}
          </p>
        )}

        {/* Stats */}
        <div className="flex justify-center gap-8 mt-4">
          <div className="text-center">
            <p className="text-lg font-semibold tabular-nums">
              {profile.content_count}
            </p>
            <p className="text-xs text-[var(--color-text-muted)]">moments</p>
          </div>
          <div className="text-center">
            <p className="text-lg font-semibold tabular-nums">
              {profile.trust_profile?.helped_count ?? 0}
            </p>
            <p className="text-xs text-[var(--color-text-muted)]">helped</p>
          </div>
          <div className="text-center">
            <p className="text-lg font-semibold tabular-nums">
              {(profile.trust_profile?.response_quality ?? 0).toFixed(1)}
            </p>
            <p className="text-xs text-[var(--color-text-muted)]">quality</p>
          </div>
        </div>

        {/* Actions */}
        <div className="mt-4 flex justify-center gap-3">
          {!isMe && (
            <FollowButton
              userId={profile.id}
              initialFollowing={profile.is_following}
            />
          )}
          {isMe && (
            <Link
              to="/settings"
              className="inline-flex items-center rounded-[var(--radius-sm)] bg-[var(--color-surface)] px-4 py-2 text-sm font-medium hover:bg-[var(--color-surface-hover)] transition-colors"
            >
              Edit profile
            </Link>
          )}
        </div>
      </section>

      {/* Trust profile */}
      {profile.trust_profile && (
        <section className="rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border)] p-4 space-y-3">
          <h3 className="text-[13px] font-semibold text-[var(--color-text-muted)] uppercase tracking-wide">
            Expertise
          </h3>
          {profile.trust_profile.topics && (
            <p className="text-sm">{profile.trust_profile.topics}</p>
          )}
          {profile.trust_profile.lived_experience && (
            <p className="text-sm text-[var(--color-text-secondary)] leading-relaxed">
              {profile.trust_profile.lived_experience}
            </p>
          )}
          <p className="text-xs text-[var(--color-text-muted)]">
            {profile.trust_profile.availability.replace("_", " ")}
          </p>
        </section>
      )}

      {/* Content grid */}
      <section className="space-y-3">
        <h3 className="text-[13px] font-semibold text-[var(--color-text-muted)] uppercase tracking-wide">
          Moments
        </h3>
        {content.length === 0 ? (
          <p className="text-sm text-[var(--color-text-muted)] text-center py-8">
            No content yet.
          </p>
        ) : (
          <div className="grid grid-cols-3 gap-1">
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
        <ContentModal
          content={selected}
          onClose={() => setSelected(null)}
          onDelete={() => {
            setContent((prev) => prev.filter((c) => c.id !== selected.id));
            setSelected(null);
          }}
        />
      )}
    </div>
  );
}
