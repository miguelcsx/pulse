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
  { value: "image", label: "Image" },
  { value: "video", label: "Video" },
  { value: "short_video", label: "Short Video" },
  { value: "text", label: "Text Post" },
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
  usePageTitle("Create Post");
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

        setPipelineStatus("Initializing upload session...");
        const { asset, upload } = await initMediaUpload(contentType, file);

        setPipelineStatus("Uploading media...");
        await uploadMediaBinary(upload.url, file, setUploadProgress);

        setPipelineStatus("Processing media...");
        const readyAsset = await waitForReady(asset.id);
        await uploadContent(contentType, null, body, tags, readyAsset.id);
      }

      addToast("Posted!", "success");
      navigate("/");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to post";
      addToast(message, "error");
    } finally {
      setLoading(false);
      setUploadProgress(0);
      setPipelineStatus("");
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <h2 className="text-lg font-semibold">Create Post</h2>

      <div className="flex gap-2">
        {contentTypes.map((ct) => (
          <button
            key={ct.value}
            type="button"
            onClick={() => {
              setContentType(ct.value);
              if (ct.value === "text") setFile(null);
            }}
            className={`text-xs px-3 py-1.5 rounded-full border transition-colors ${
              contentType === ct.value
                ? "bg-indigo-600 border-indigo-600 text-white"
                : "border-[var(--color-border)] text-[var(--color-text-muted)] hover:border-indigo-500"
            }`}
          >
            {ct.label}
          </button>
        ))}
      </div>

      {needsFile && (
        <label className="block border-2 border-dashed border-[var(--color-border)] rounded-lg p-8 text-center cursor-pointer hover:border-indigo-500 transition-colors">
          {preview ? (
            contentType === "image" ? (
              <img
                src={preview}
                alt="Preview"
                className="max-h-64 mx-auto rounded"
              />
            ) : (
              <video
                src={preview}
                className="max-h-64 mx-auto rounded"
                controls
              />
            )
          ) : (
            <div className="text-[var(--color-text-muted)]">
              <p>Click to select a file</p>
              <p className="text-xs mt-1">
                {contentType === "image" ? "JPG, PNG, WebP" : "MP4, MOV, WebM"}{" "}
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

      <div>
        <label className="block text-sm font-medium mb-1">
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
          className="w-full px-3 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] text-sm text-[var(--color-text)] placeholder-[var(--color-text-muted)]"
        />
      </div>

      <div>
        <label className="block text-sm font-medium mb-1">Tags</label>
        <div className="flex gap-2">
          <input
            type="text"
            value={tagInput}
            onChange={(e) => setTagInput(e.target.value)}
            onKeyDown={handleTagKeyDown}
            onBlur={addTag}
            placeholder="Type a tag and press Enter..."
            className="flex-1 px-3 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] text-sm text-[var(--color-text)] placeholder-[var(--color-text-muted)]"
          />
        </div>
        {tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mt-2">
            {tags.map((tag) => (
              <button
                key={tag}
                type="button"
                onClick={() => removeTag(tag)}
                className="text-xs px-2 py-1 rounded-full bg-indigo-600 text-white hover:bg-indigo-500"
              >
                #{tag} &times;
              </button>
            ))}
          </div>
        )}
      </div>

      {loading && pipelineStatus && (
        <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
          <p className="text-xs text-[var(--color-text-muted)]">
            {pipelineStatus}
          </p>
          {uploadProgress > 0 && uploadProgress < 100 && (
            <div className="mt-2 h-2 w-full rounded-full bg-[var(--color-surface-hover)]">
              <div
                className="h-2 rounded-full bg-indigo-500 transition-all"
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
      >
        Post
      </Button>
    </form>
  );
}
