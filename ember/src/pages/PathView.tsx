import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { getPath, followPath, unfollowPath, createPath } from "../api/paths";
import { getUserContent } from "../api/content";
import { trackPathFollow, trackView } from "../api/events";
import { useUiStore } from "../store/uiStore";
import { useAuthStore } from "../store/authStore";
import { usePageTitle } from "../hooks/usePageTitle";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import Spinner from "../components/ui/Spinner";
import MediaFallback from "../components/ui/MediaFallback";
import type { Content, Path } from "@pulse/drift/types";

interface SelectedPathItem {
  content_id: string;
  note: string;
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
      return (
        <MediaFallback
          src={content.media_url}
          alt={content.body || "Video content"}
          type={content.content_type}
          controls
          className={className}
          preload="metadata"
        />
      );
    case "text":
      return (
        <div className="p-3 bg-[var(--color-surface-hover)] rounded border border-[var(--color-border)]">
          <p className="text-sm line-clamp-5 whitespace-pre-wrap">
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
  const isNew = !id;

  const [path, setPath] = useState<Path | null>(null);
  const [loading, setLoading] = useState(!isNew);
  usePageTitle(isNew ? "Create Path" : path?.title ? path.title : "Path");

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [availableContent, setAvailableContent] = useState<Content[]>([]);
  const [selectedItems, setSelectedItems] = useState<SelectedPathItem[]>([]);
  const [loadingContent, setLoadingContent] = useState(isNew);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    if (id) {
      getPath(id)
        .then((data) => {
          setPath(data);
          trackView("path", id);
        })
        .catch(() => addToast("Path not found", "error"))
        .finally(() => setLoading(false));
    }
  }, [id, addToast]);

  useEffect(() => {
    if (!isNew || !user?.id) {
      setLoadingContent(false);
      return;
    }

    getUserContent(user.id, undefined, 50)
      .then((res) => setAvailableContent(res.items))
      .catch(() => addToast("Failed to load your content", "error"))
      .finally(() => setLoadingContent(false));
  }, [isNew, user?.id, addToast]);

  const toggleItem = (contentId: string) => {
    setSelectedItems((prev) => {
      const exists = prev.some((item) => item.content_id === contentId);
      if (exists) {
        return prev.filter((item) => item.content_id !== contentId);
      }
      return [...prev, { content_id: contentId, note: "" }];
    });
  };

  const updateItemNote = (contentId: string, note: string) => {
    setSelectedItems((prev) =>
      prev.map((item) =>
        item.content_id === contentId ? { ...item, note } : item,
      ),
    );
  };

  const moveItem = (index: number, direction: "up" | "down") => {
    setSelectedItems((prev) => {
      const next = [...prev];
      const target = direction === "up" ? index - 1 : index + 1;
      if (target < 0 || target >= next.length) {
        return prev;
      }
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
  };

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

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim() || selectedItems.length === 0) return;

    setCreating(true);
    try {
      const newPath = await createPath({
        title: title.trim(),
        description: description.trim(),
        items: selectedItems,
      });
      addToast("Path created", "success");
      navigate(`/paths/${newPath.id}`);
    } catch {
      addToast("Failed to create path", "error");
    } finally {
      setCreating(false);
    }
  };

  if (isNew) {
    return (
      <form onSubmit={handleCreate} className="space-y-5">
        <h2 className="text-lg font-semibold">Create Path</h2>
        <Input
          label="Title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Give your path a name"
          required
        />
        <div>
          <label className="block text-sm font-medium mb-1">Description</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="What's this path about?"
            rows={3}
            className="w-full px-3 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] text-sm text-[var(--color-text)] placeholder-[var(--color-text-muted)]"
          />
        </div>

        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">
              Selected items ({selectedItems.length})
            </h3>
          </div>
          {selectedItems.length === 0 ? (
            <p className="text-sm text-[var(--color-text-muted)]">
              Pick content below to build your path.
            </p>
          ) : (
            <div className="space-y-3">
              {selectedItems.map((item, index) => {
                const content = availableContent.find(
                  (c) => c.id === item.content_id,
                );
                return (
                  <div
                    key={item.content_id}
                    className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-3 space-y-2"
                  >
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-[var(--color-text-muted)]">
                        Step {index + 1}
                      </span>
                      <div className="flex gap-2">
                        <Button
                          type="button"
                          size="sm"
                          variant="secondary"
                          onClick={() => moveItem(index, "up")}
                        >
                          Up
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="secondary"
                          onClick={() => moveItem(index, "down")}
                        >
                          Down
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="danger"
                          onClick={() => toggleItem(item.content_id)}
                        >
                          Remove
                        </Button>
                      </div>
                    </div>
                    {content &&
                      renderContentPreview(
                        content,
                        "w-full max-h-48 object-cover rounded",
                      )}
                    <input
                      type="text"
                      value={item.note}
                      onChange={(e) =>
                        updateItemNote(item.content_id, e.target.value)
                      }
                      placeholder="Optional note for this step"
                      className="w-full px-3 py-2 rounded-lg bg-[var(--color-surface-hover)] border border-[var(--color-border)] text-sm"
                    />
                  </div>
                );
              })}
            </div>
          )}
        </section>

        <section className="space-y-3">
          <h3 className="text-sm font-semibold">Your content</h3>
          {loadingContent ? (
            <div className="flex justify-center py-8">
              <Spinner size="md" />
            </div>
          ) : availableContent.length === 0 ? (
            <p className="text-sm text-[var(--color-text-muted)]">
              You need to post content before creating a path.
            </p>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {availableContent.map((content) => {
                const selected = selectedItems.some(
                  (item) => item.content_id === content.id,
                );
                return (
                  <div
                    key={content.id}
                    className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-3 space-y-2"
                  >
                    {renderContentPreview(
                      content,
                      "w-full h-40 object-cover rounded",
                    )}
                    <div className="flex items-center justify-between">
                      <p className="text-xs text-[var(--color-text-muted)]">
                        {content.content_type.replace("_", " ")}
                      </p>
                      <Button
                        type="button"
                        size="sm"
                        variant={selected ? "secondary" : "primary"}
                        onClick={() => toggleItem(content.id)}
                      >
                        {selected ? "Remove" : "Add"}
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </section>

        <Button
          type="submit"
          loading={creating}
          disabled={!title.trim() || selectedItems.length === 0}
        >
          Create Path
        </Button>
      </form>
    );
  }

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    );
  }

  if (!path) {
    return <p className="text-[var(--color-text-muted)]">Path not found</p>;
  }

  const isOwner = user?.id === path.creator_id;

  return (
    <div className="space-y-6">
      <button
        onClick={() => navigate("/paths")}
        className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
      >
        &larr; Back to Paths
      </button>

      <div>
        <h2 className="text-xl font-semibold">{path.title}</h2>
        {path.description && (
          <p className="text-sm text-[var(--color-text-muted)] mt-1">
            {path.description}
          </p>
        )}
        <div className="flex items-center gap-4 mt-2 text-xs text-[var(--color-text-muted)]">
          <span>by {path.creator?.display_name}</span>
          <span>{path.follower_count} followers</span>
        </div>
        {!isOwner &&
          (path.is_following ? (
            <Button
              size="sm"
              variant="secondary"
              className="mt-3"
              onClick={handleUnfollow}
            >
              Unfollow Path
            </Button>
          ) : (
            <Button size="sm" className="mt-3" onClick={handleFollow}>
              Follow Path
            </Button>
          ))}
      </div>

      <div className="space-y-3">
        {path.items?.length === 0 ? (
          <p className="text-[var(--color-text-muted)] text-sm">
            No items in this path yet.
          </p>
        ) : (
          path.items?.map((item, i) => (
            <div
              key={item.id}
              className="bg-[var(--color-surface)] rounded-lg overflow-hidden border border-[var(--color-border)]"
            >
              <div className="flex items-center gap-3 p-3">
                <span className="text-xs font-mono text-[var(--color-text-muted)]">
                  {i + 1}
                </span>
                {item.note && <p className="text-sm">{item.note}</p>}
              </div>
              {item.content &&
                renderContentPreview(
                  item.content,
                  "w-full aspect-video object-cover",
                )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}
