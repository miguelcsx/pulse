import { useCallback, useEffect, useRef } from "react";
import type { AffinityFeedItem, FeedMoment } from "@pulse/drift/types";
import { useFeedContextStore } from "../store/feedContextStore";

export function useVisibleRoomContext(items: AffinityFeedItem[]) {
  const setActiveRoom = useFeedContextStore((s) => s.setActiveRoom);
  const visibleIds = useRef(new Set<string>());
  const observerRef = useRef<IntersectionObserver | null>(null);
  const itemsRef = useRef(items);
  useEffect(() => {
    itemsRef.current = items;
  }, [items]);

  const computeDominantRoom = useCallback(() => {
    const roomCounts = new Map<string, { count: number; item: FeedMoment }>();

    for (const id of visibleIds.current) {
      const unit = itemsRef.current.find((i) => i.id === id);
      const item = unit?.content
        ? { ...unit.content, room_context: unit.room_context }
        : undefined;
      if (!item?.room_context) continue;

      const roomId = item.room_context.room_id;
      const existing = roomCounts.get(roomId);
      if (existing) {
        existing.count++;
      } else {
        roomCounts.set(roomId, { count: 1, item });
      }
    }

    let dominant: FeedMoment | null = null;
    let maxCount = 0;
    for (const [, { count, item }] of roomCounts) {
      if (count >= 2 && count > maxCount) {
        maxCount = count;
        dominant = item;
      }
    }

    setActiveRoom(dominant?.room_context ?? null);
  }, [setActiveRoom]);

  useEffect(() => {
    observerRef.current = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          const contentId = (entry.target as HTMLElement).dataset.contentId;
          if (!contentId) continue;
          if (entry.isIntersecting) {
            visibleIds.current.add(contentId);
          } else {
            visibleIds.current.delete(contentId);
          }
        }
        computeDominantRoom();
      },
      { threshold: 0.5 },
    );

    return () => {
      observerRef.current?.disconnect();
    };
  }, [computeDominantRoom]);

  const observe = useCallback((el: HTMLElement | null) => {
    if (el && observerRef.current) {
      observerRef.current.observe(el);
    }
  }, []);

  return observe;
}
