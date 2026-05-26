import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  getMediaAsset,
  initMediaUpload,
  uploadContent,
  uploadMediaBinary,
} from "../api/content";
import { runtimeConfig } from "../config/runtime";
import { useUiStore } from "../store/uiStore";
import { usePageTitle } from "../hooks/usePageTitle";
import Button from "../components/ui/Button";
import type { ContentType, MediaAsset } from "@pulse/drift/types";

const contentTypes: { value: ContentType; label: string }[] = [
  { value: "image", label: "Photo" },
  { value: "video", label: "Video" },
  { value: "short_video", label: "Short" },
  { value: "text", label: "Text" },
];

async function waitForReady(
  assetId: string,
  timeoutMs: number = runtimeConfig.mediaReadyTimeoutMs,
  pollIntervalMs: number = runtimeConfig.mediaReadyPollIntervalMs,
): Promise<MediaAsset> {
  const startedAt = Date.now();

  while (Date.now() - startedAt < timeoutMs) {
    const asset = await getMediaAsset(assetId);
    if (asset.status === "ready") {
      return asset;
    }
    if (asset.status === "failed") {
      throw new Error(asset.error_message || "Media processing failed");
    }
    await new Promise((resolve) => setTimeout(resolve, pollIntervalMs));
  }

  throw new Error("Media processing timed out");
}

export default function Upload() {
  usePageTitle("New post");
  const navigate = useNavigate();
  const addToast = useUiStore((s) => s.addToast);
  const [contentType, setContentType] = useState<ContentType>("image");
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState("");
  const [body, setBody] = useState("");
  const [tagInput, setTagInput] = useState("");
  const [tags, setTags] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [pipelineStatus, setPipelineStatus] = useState("");

  const needsFile = contentType !== "text";

  useEffect(() => {
    if (!file) {
      setPreview("");
      return;
    }
    const url = URL.createObjectURL(file);
    setPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);

  const addTag = () => {
    const raw = tagInput.trim().toLowerCase().replace(/^#/, "");
    if (raw && !tags.includes(raw)) {
      setTags([...tags, raw]);
    }
    setTagInput("");
  };

  const handleTagKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" || e.key === "," || e.key === " ") {
      e.preventDefault();
      addTag();
    }
  };

  const removeTag = (tag: string) => {
    setTags(tags.filter((t) => t !== tag));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (needsFile && !file) return;
    if (contentType === "text" && !body.trim()) return;

    setLoading(true);
    setUploadProgress(0);
    setPipelineStatus("");

    try {
      if (contentType === "text") {
        await uploadContent(contentType, null, body, tags);
      } else {
        if (!file) return;

        setPipelineStatus("Initializing...");
        const { asset, upload } = await initMediaUpload(contentType, file);

        setPipelineStatus("Uploading...");
        await uploadMediaBinary(upload.url, file, setUploadProgress);

        setPipelineStatus("Processing...");
        const readyAsset = await waitForReady(asset.id);
        await uploadContent(contentType, null, body, tags, readyAsset.id);
      }

      addToast("Posted!", "success");
      navigate("/moments");
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to post";
      addToast(message, "error");
    } finally {
      setLoading(false);
      setUploadProgress(0);
      setPipelineStatus("");
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5 pb-4">
      <h2 className="text-[22px] font-semibold tracking-tight pt-2">
        New post
      </h2>

      {/* Type selector */}
      <div className="flex gap-2">
        {contentTypes.map((ct) => (
          <button
            key={ct.value}
            type="button"
            onClick={() => {
              setContentType(ct.value);
              if (ct.value === "text") setFile(null);
            }}
            className={`text-[13px] font-medium px-4 py-2 rounded-full transition-all ${
              contentType === ct.value
                ? "bg-[var(--color-primary)] text-[var(--color-bg)] shadow-sm"
                : "bg-[var(--color-surface)] text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
            }`}
          >
            {ct.label}
          </button>
        ))}
      </div>

      {/* File drop zone */}
      {needsFile && (
        <label className="block rounded-[var(--radius-lg)] border-2 border-dashed border-[var(--color-border)] p-8 text-center cursor-pointer hover:border-[var(--color-accent)] transition-colors">
          {preview ? (
            contentType === "image" ? (
              <img
                src={preview}
                alt="Preview"
                className="max-h-64 mx-auto rounded-[var(--radius-sm)]"
              />
            ) : (
              <video
                src={preview}
                className="max-h-64 mx-auto rounded-[var(--radius-sm)]"
                controls
              />
            )
          ) : (
            <div className="text-[var(--color-text-muted)] space-y-1">
              <div className="mx-auto w-10 h-10 rounded-full bg-[var(--color-surface)] flex items-center justify-center mb-3">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4" />
                  <polyline points="17 8 12 3 7 8" />
                  <line x1="12" y1="3" x2="12" y2="15" />
                </svg>
              </div>
              <p className="text-sm font-medium">Tap to select</p>
              <p className="text-xs">
                {contentType === "image"
                  ? "JPG, PNG, WebP"
                  : "MP4, MOV, WebM"}{" "}
                up to 10MB
              </p>
            </div>
          )}
          <input
            type="file"
            accept={contentType === "image" ? "image/*" : "video/*"}
            className="hidden"
            onChange={(e) => setFile(e.target.files?.[0] || null)}
          />
        </label>
      )}

      {/* Caption */}
      <div>
        <label className="block text-[13px] font-medium text-[var(--color-text-secondary)] mb-1.5">
          {contentType === "text" ? "What's on your mind?" : "Caption"}
        </label>
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder={
            contentType === "text"
              ? "Share your thoughts..."
              : "What's the story behind this?"
          }
          rows={contentType === "text" ? 5 : 2}
          className="w-full px-3 py-2.5 rounded-[var(--radius-sm)] bg-[var(--color-bg)] border border-[var(--color-border)] text-sm text-[var(--color-text)] placeholder-[var(--color-text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-colors"
        />
      </div>

      {/* Tags */}
      <div>
        <label className="block text-[13px] font-medium text-[var(--color-text-secondary)] mb-1.5">
          Tags
        </label>
        <input
          type="text"
          value={tagInput}
          onChange={(e) => setTagInput(e.target.value)}
          onKeyDown={handleTagKeyDown}
          onBlur={addTag}
          placeholder="Type a tag and press Enter..."
          className="w-full px-3 py-2.5 rounded-[var(--radius-sm)] bg-[var(--color-bg)] border border-[var(--color-border)] text-sm text-[var(--color-text)] placeholder-[var(--color-text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent transition-colors"
        />
        {tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mt-2">
            {tags.map((tag) => (
              <button
                key={tag}
                type="button"
                onClick={() => removeTag(tag)}
                className="text-xs px-2.5 py-1 rounded-full bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity"
              >
                #{tag} &times;
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Progress */}
      {loading && pipelineStatus && (
        <div className="rounded-[var(--radius-sm)] bg-[var(--color-surface)] p-3">
          <p className="text-xs text-[var(--color-text-muted)]">
            {pipelineStatus}
          </p>
          {uploadProgress > 0 && uploadProgress < 100 && (
            <div className="mt-2 h-1 w-full rounded-full bg-[var(--color-surface-hover)]">
              <div
                className="h-1 rounded-full bg-[var(--color-accent)] transition-all"
                style={{ width: `${uploadProgress}%` }}
              />
            </div>
          )}
        </div>
      )}

      <Button
        type="submit"
        loading={loading}
        disabled={needsFile ? !file : !body.trim()}
        className="w-full"
      >
        Post
      </Button>
    </form>
  );
}
