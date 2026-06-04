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
    <div className="space-y-6 pb-4">
      <button
        onClick={() => navigate(-1)}
        className="flex items-center gap-1 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors pt-2"
      >
        <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <polyline points="15 18 9 12 15 6" />
        </svg>
        Back
      </button>

      <section className="rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border)] p-5">
        <h2 className="text-xl font-semibold mb-2">Shared context</h2>
        <p className="mb-3 text-sm text-[var(--color-text-muted)]">
          Related moments from people posting around the same tags. Joining only
          marks that you are exploring this context.
        </p>
        <PresenceIndicator count={presence} />
        {error && (
          <p className="mt-3 text-sm text-[var(--color-error)]">{error}</p>
        )}

        <div className="mt-5">
          {joined ? (
            <div className="space-y-3">
              <p className="flex items-center gap-2 text-sm text-[var(--color-success)]">
                <span className="h-2 w-2 rounded-full bg-[var(--color-success)]" />
                You&rsquo;re exploring this context
              </p>
              <Button variant="danger" onClick={handleLeave} loading={loading}>
                Leave context
              </Button>
            </div>
          ) : (
            <Button variant="accent" onClick={handleEnter} loading={loading}>
              Explore context
            </Button>
          )}
        </div>
      </section>

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-[17px] font-semibold">Related moments</h3>
          {roomContent.length > 0 && (
            <span className="text-xs text-[var(--color-text-muted)] tabular-nums">
              {roomContent.length} posts
            </span>
          )}
        </div>

        {contentLoading ? (
          <div className="flex justify-center py-8">
            <Spinner />
          </div>
        ) : contentError ? (
          <p className="text-sm text-[var(--color-error)]">{contentError}</p>
        ) : roomContent.length === 0 ? (
          <p className="text-sm text-[var(--color-text-muted)] text-center py-8">
            No related moments yet.
          </p>
        ) : (
          <div className="space-y-4">
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
