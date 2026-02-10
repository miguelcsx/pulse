import { useState, useEffect, useCallback } from "react";
import { Link } from "react-router-dom";
import { getRooms } from "../api/rooms";
import Spinner from "../components/ui/Spinner";
import { useUiStore } from "../store/uiStore";
import { usePageTitle } from "../hooks/usePageTitle";
import type { Room } from "@pulse/drift/types";

export default function Rooms() {
  usePageTitle("Rooms");
  const [rooms, setRooms] = useState<Room[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [retryCount, setRetryCount] = useState(0);
  const addToast = useUiStore((s) => s.addToast);

  useEffect(() => {
    let cancelled = false;

    getRooms()
      .then((data) => {
        if (!cancelled) {
          setRooms(data);
          setLoading(false);
          setError(null);
        }
      })
      .catch(() => {
        if (!cancelled) {
          const msg = "Failed to load rooms";
          setError(msg);
          setLoading(false);
          addToast(msg, "error");
        }
      });

    return () => {
      cancelled = true;
    };
  }, [addToast, retryCount]);

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

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Mood Rooms</h2>
      {rooms.length === 0 ? (
        <p className="text-[var(--color-text-muted)] text-sm">
          No active rooms right now. Post with tags to create rooms.
        </p>
      ) : (
        rooms.map((room) => (
          <Link
            key={room.id}
            to={`/rooms/${room.id}`}
            className="block bg-[var(--color-surface)] rounded-lg p-4 border border-[var(--color-border)] hover:border-indigo-500 transition-colors"
          >
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-medium">
                {room.tags?.map((t) => t.name).join(" + ") || "Room"}
              </span>
              <span className="text-xs text-[var(--color-text-muted)]">
                {room.member_count}{" "}
                {room.member_count === 1 ? "person" : "people"}
              </span>
            </div>
            <div className="flex flex-wrap gap-1.5">
              {room.tags?.map((tag) => (
                <span
                  key={tag.id}
                  className="text-xs px-2 py-0.5 rounded-full bg-indigo-900/30 text-indigo-300"
                >
                  {tag.name}
                </span>
              ))}
            </div>
          </Link>
        ))
      )}
    </div>
  );
}
