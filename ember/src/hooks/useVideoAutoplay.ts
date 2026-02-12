import { useCallback, useRef } from "react";

interface Options {
  threshold?: number;
}

export function useVideoAutoplay(
  options: Options = {},
): React.RefCallback<HTMLVideoElement> {
  const { threshold = 0.6 } = options;
  const observerRef = useRef<IntersectionObserver | null>(null);
  const videoRef = useRef<HTMLVideoElement | null>(null);

  const refCallback = useCallback(
    (node: HTMLVideoElement | null) => {
      // Cleanup previous observer
      if (observerRef.current) {
        observerRef.current.disconnect();
        observerRef.current = null;
      }

      videoRef.current = node;

      if (!node) return;

      observerRef.current = new IntersectionObserver(
        (entries) => {
          for (const entry of entries) {
            const video = entry.target as HTMLVideoElement;
            if (entry.isIntersecting) {
              video.play().catch(() => {
                // Browser blocked autoplay — expected if user hasn't interacted
              });
            } else {
              video.pause();
            }
          }
        },
        { threshold },
      );

      observerRef.current.observe(node);
    },
    [threshold],
  );

  return refCallback;
}
