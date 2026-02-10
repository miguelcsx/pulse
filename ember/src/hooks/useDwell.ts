import { useEffect, useRef, useCallback } from "react";
import { trackDwell } from "../api/events";

/**
 * Tracks how long an element is visible in the viewport.
 * When the element leaves the viewport (or the component unmounts),
 * a "dwell" event is recorded with the accumulated visible duration.
 *
 * Used on feed cards and content views to power the affinity graph's
 * dwell signal — longer dwell = stronger interest signal.
 */
export function useDwell(
  targetType: string,
  targetId: string | undefined,
  /** Minimum dwell time (ms) before recording an event. Avoids noise from quick scrolls. */
  thresholdMs: number = 1000,
): React.RefCallback<HTMLElement> {
  const entryTimeRef = useRef<number | null>(null);
  const accumulatedRef = useRef<number>(0);
  const observerRef = useRef<IntersectionObserver | null>(null);
  const elementRef = useRef<HTMLElement | null>(null);
  const targetIdRef = useRef(targetId);
  const targetTypeRef = useRef(targetType);

  // Keep refs in sync via effect rather than during render
  useEffect(() => {
    targetIdRef.current = targetId;
  }, [targetId]);

  useEffect(() => {
    targetTypeRef.current = targetType;
  }, [targetType]);

  useEffect(() => {
    return () => {
      // Flush any accumulated dwell time on unmount
      const id = targetIdRef.current;
      if (!id) return;

      let total = accumulatedRef.current;
      if (entryTimeRef.current !== null) {
        total += Date.now() - entryTimeRef.current;
        entryTimeRef.current = null;
      }

      if (total >= thresholdMs) {
        trackDwell(targetTypeRef.current, id, Math.round(total));
      }

      accumulatedRef.current = 0;
    };
  }, [thresholdMs]);

  // Ref callback — called when the DOM element is attached/detached
  const setRef = useCallback(
    (node: HTMLElement | null) => {
      // Clean up previous observer
      if (observerRef.current) {
        observerRef.current.disconnect();
        observerRef.current = null;
      }

      // If leaving viewport while swapping elements, accumulate time
      if (entryTimeRef.current !== null) {
        accumulatedRef.current += Date.now() - entryTimeRef.current;
        entryTimeRef.current = null;
      }

      elementRef.current = node;

      if (!node || !targetId) return;

      const observer = new IntersectionObserver(
        (entries) => {
          for (const entry of entries) {
            if (entry.isIntersecting) {
              // Element entered viewport
              if (entryTimeRef.current === null) {
                entryTimeRef.current = Date.now();
              }
            } else {
              // Element left viewport — accumulate duration
              if (entryTimeRef.current !== null) {
                accumulatedRef.current += Date.now() - entryTimeRef.current;
                entryTimeRef.current = null;
              }
            }
          }
        },
        { threshold: 0.5 },
      );

      observer.observe(node);
      observerRef.current = observer;
    },
    [targetId],
  );

  return setRef;
}
