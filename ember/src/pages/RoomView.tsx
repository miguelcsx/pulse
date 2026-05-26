import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { enterRoom, leaveRoom, getRoomContent } from "../api/rooms";
import type { Content } from "@pulse/drift/types";
import { useWsStore } from "../store/wsStore";
import { usePageTitle } from "../hooks/usePageTitle";
import FeedCard from "../components/feed/FeedCard";
import ContentModal from "../components/feed/ContentModal";
import Button from "../components/ui/Button";
import Spinner from "../components/ui/Spinner";
import PresenceIndicator from "../components/rooms/PresenceIndicator";

export default function RoomView() {
  usePageTitle("Room");
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [joined, setJoined] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [roomContent, setRoomContent] = useState<Content[]>([]);
  const [contentLoading, setContentLoading] = useState(true);
  const [contentError, setContentError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Content | null>(null);
  const presence = useWsStore((s) => s.roomPresence[id || ""] || 0);
  const updatePresence = useWsStore((s) => s.updatePresence);
  const joinRoom = useWsStore((s) => s.joinRoom);
  const wsLeaveRoom = useWsStore((s) => s.leaveRoom);

  const handleEnter = async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const memberCount = await enterRoom(id);
      updatePresence(id, memberCount);
      joinRoom(id);
      setJoined(true);
    } catch {
      setError("Could not enter this room. It may be expired.");
    } finally {
      setLoading(false);
    }
  };

  const handleLeave = async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const memberCount = await leaveRoom(id);
      updatePresence(id, memberCount);
      wsLeaveRoom(id);
      setJoined(false);
    } catch {
      setError("Could not leave this room. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    setContentLoading(true);
    setContentError(null);

    getRoomContent(id)
      .then((data) => {
        if (!cancelled) {
          setRoomContent(data);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setContentError("Failed to load room content.");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setContentLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [id]);

  useEffect(() => {
    return () => {
      if (joined && id) {
        leaveRoom(id).catch(() => {});
        wsLeaveRoom(id);
      }
    };
  }, [joined, id, wsLeaveRoom]);

  return (
    <div className="space-y-6">
      <button
        onClick={() => navigate("/discover")}
        className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
      >
        &larr; Back to Discover
      </button>

      <div className="bg-[var(--color-surface)] rounded-lg p-6 border border-[var(--color-border)]">
        <h2 className="text-lg font-semibold mb-4">Room</h2>
        <PresenceIndicator count={presence} />
        {error ? (
          <p className="mt-3 text-sm text-[var(--color-error)]">{error}</p>
        ) : null}

        <div className="mt-6">
          {joined ? (
            <div className="space-y-4">
              <p className="text-sm text-[var(--color-success)]">
                You're in this room
              </p>
              <Button variant="danger" onClick={handleLeave} loading={loading}>
                Leave Room
              </Button>
            </div>
          ) : (
            <Button onClick={handleEnter} loading={loading}>
              Enter Room
            </Button>
          )}
        </div>
      </div>

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold">Room feed</h3>
          {roomContent.length > 0 && (
            <span className="text-xs text-[var(--color-text-muted)]">
              {roomContent.length} posts
            </span>
          )}
        </div>

        {contentLoading ? (
          <Spinner />
        ) : contentError ? (
          <p className="text-sm text-[var(--color-error)]">{contentError}</p>
        ) : roomContent.length === 0 ? (
          <p className="text-sm text-[var(--color-text-muted)]">
            No content in this room yet.
          </p>
        ) : (
          <div className="space-y-6">
            {roomContent.map((item) => (
              <FeedCard
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
