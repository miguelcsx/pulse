import { useState } from "react";

interface MediaFallbackProps {
  src: string | undefined;
  alt?: string;
  type: "image" | "video" | "short_video";
  className?: string;
  controls?: boolean;
  muted?: boolean;
  playsInline?: boolean;
  preload?: string;
  loading?: "lazy" | "eager";
  autoPlay?: boolean;
  videoRef?: React.RefCallback<HTMLVideoElement>;
}

function Placeholder({ className, message }: { className?: string; message: string }) {
  return (
    <div
      className={`flex flex-col items-center justify-center gap-2 bg-[var(--color-surface-hover)] text-[var(--color-text-muted)] ${className ?? ""}`}
      role="img"
      aria-label={message}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="h-8 w-8 opacity-40"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
        <line x1="9" y1="3" x2="9" y2="21" />
        <line x1="15" y1="3" x2="15" y2="21" />
        <line x1="3" y1="9" x2="21" y2="9" />
        <line x1="3" y1="15" x2="21" y2="15" />
      </svg>
      <span className="text-xs">{message}</span>
    </div>
  );
}

export default function MediaFallback({
  src,
  alt,
  type,
  className = "",
  controls,
  muted,
  playsInline,
  preload,
  loading,
  autoPlay,
  videoRef,
}: MediaFallbackProps) {
  const [errored, setErrored] = useState(false);

  if (!src || errored) {
    const message =
      type === "image"
        ? "Image unavailable"
        : "Video unavailable";
    return <Placeholder className={className} message={message} />;
  }

  if (type === "image") {
    return (
      <img
        src={src}
        alt={alt || "Content image"}
        className={className}
        loading={loading}
        onError={() => setErrored(true)}
      />
    );
  }

  return (
    <video
      ref={videoRef}
      src={src}
      className={className}
      controls={controls}
      muted={muted}
      playsInline={playsInline}
      preload={preload}
      autoPlay={autoPlay}
      loop
      onError={() => setErrored(true)}
    />
  );
}
