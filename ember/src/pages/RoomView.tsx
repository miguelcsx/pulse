import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { enterRoom, leaveRoom } from "../api/rooms";
import { useWsStore } from "../store/wsStore";
import { usePageTitle } from "../hooks/usePageTitle";
import Button from "../components/ui/Button";
import PresenceIndicator from "../components/rooms/PresenceIndicator";

export default function RoomView() {
  usePageTitle("Room");
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [joined, setJoined] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
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
        {error ? <p className="mt-3 text-sm text-[var(--color-error)]">{error}</p> : null}

        <div className="mt-6">
          {joined ? (
            <div className="space-y-4">
              <p className="text-sm text-[var(--color-success)]">You're in this room</p>
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
    </div>
  );
}
