import { useEffect, useRef, useCallback } from "react";
import { useAuthStore } from "../store/authStore";
import { useWsStore } from "../store/wsStore";
import { getWebSocketURL } from "../api/ws";
import { WS_MESSAGE_TYPES } from "@pulse/drift/constants";

const INITIAL_RECONNECT_DELAY_MS = 1000;
const MAX_RECONNECT_DELAY_MS = 30_000;
const RECONNECT_BACKOFF_FACTOR = 2;
const RECONNECT_JITTER_FACTOR = 0.3;
const HEARTBEAT_INTERVAL_MS = 25_000;
const HEARTBEAT_TIMEOUT_MS = 10_000;

function jitteredDelay(base: number): number {
  const jitter = base * RECONNECT_JITTER_FACTOR * (Math.random() * 2 - 1);
  return Math.max(0, Math.round(base + jitter));
}

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectDelayRef = useRef(INITIAL_RECONNECT_DELAY_MS);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const heartbeatTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pongTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const intentionalCloseRef = useRef(false);
  const mountedRef = useRef(true);

  const accessToken = useAuthStore((s) => s.accessToken);
  const setConnected = useWsStore((s) => s.setConnected);
  const updatePresence = useWsStore((s) => s.updatePresence);

  const clearTimers = useCallback(() => {
    if (reconnectTimerRef.current !== null) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    if (heartbeatTimerRef.current !== null) {
      clearInterval(heartbeatTimerRef.current);
      heartbeatTimerRef.current = null;
    }
    if (pongTimerRef.current !== null) {
      clearTimeout(pongTimerRef.current);
      pongTimerRef.current = null;
    }
  }, []);

  const closeSocket = useCallback(() => {
    const ws = wsRef.current;
    if (ws) {
      ws.onopen = null;
      ws.onclose = null;
      ws.onerror = null;
      ws.onmessage = null;
      if (
        ws.readyState === WebSocket.OPEN ||
        ws.readyState === WebSocket.CONNECTING
      ) {
        ws.close(1000, "client teardown");
      }
      wsRef.current = null;
    }
  }, []);

  const startHeartbeat = useCallback((ws: WebSocket) => {
    if (heartbeatTimerRef.current !== null) {
      clearInterval(heartbeatTimerRef.current);
    }

    heartbeatTimerRef.current = setInterval(() => {
      if (ws.readyState !== WebSocket.OPEN) return;

      try {
        ws.send(JSON.stringify({ type: "ping" }));
      } catch {
        // Socket may have closed between the check and the send
        return;
      }

      // If we don't receive any message within the timeout, consider the
      // connection dead and force-close so reconnection kicks in.
      if (pongTimerRef.current !== null) {
        clearTimeout(pongTimerRef.current);
      }
      pongTimerRef.current = setTimeout(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.close(4000, "heartbeat timeout");
        }
      }, HEARTBEAT_TIMEOUT_MS);
    }, HEARTBEAT_INTERVAL_MS);
  }, []);

  const connect = useCallback(
    (token: string) => {
      if (!mountedRef.current) return;

      closeSocket();
      clearTimers();

      const url = getWebSocketURL();
      let ws: WebSocket;
      try {
        ws = new WebSocket(url, ["bearer", token]);
      } catch {
        // Schedule a reconnect if WebSocket construction fails (bad URL, etc.)
        scheduleReconnect();
        return;
      }

      wsRef.current = ws;
      intentionalCloseRef.current = false;

      ws.onopen = () => {
        if (!mountedRef.current) return;
        reconnectDelayRef.current = INITIAL_RECONNECT_DELAY_MS;
        setConnected(true);
        startHeartbeat(ws);
      };

      ws.onmessage = (event) => {
        // Any incoming message proves the connection is alive — clear pong timeout
        if (pongTimerRef.current !== null) {
          clearTimeout(pongTimerRef.current);
          pongTimerRef.current = null;
        }

        try {
          const msg = JSON.parse(event.data);
          switch (msg.type) {
            case WS_MESSAGE_TYPES.ROOM_PRESENCE:
            case WS_MESSAGE_TYPES.USER_JOINED:
            case WS_MESSAGE_TYPES.USER_LEFT:
              if (msg.room_id && typeof msg.member_count === "number") {
                updatePresence(msg.room_id, msg.member_count);
              }
              break;
            // "pong" and unknown types are intentionally ignored
          }
        } catch {
          // Ignore malformed messages
        }
      };

      ws.onclose = () => {
        if (!mountedRef.current) return;
        setConnected(false);
        clearTimers();

        if (!intentionalCloseRef.current) {
          scheduleReconnect();
        }
      };

      ws.onerror = () => {
        // onclose fires after onerror, so reconnection is handled there.
        // Just update state immediately for faster UI feedback.
        if (mountedRef.current) {
          setConnected(false);
        }
      };
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [closeSocket, clearTimers, setConnected, updatePresence, startHeartbeat],
  );

  const scheduleReconnect = useCallback(() => {
    if (!mountedRef.current || intentionalCloseRef.current) return;

    const delay = jitteredDelay(reconnectDelayRef.current);
    reconnectDelayRef.current = Math.min(
      reconnectDelayRef.current * RECONNECT_BACKOFF_FACTOR,
      MAX_RECONNECT_DELAY_MS,
    );

    reconnectTimerRef.current = setTimeout(() => {
      // Re-read the token from store in case it was refreshed during the delay
      const currentToken = useAuthStore.getState().accessToken;
      if (currentToken && mountedRef.current) {
        connect(currentToken);
      }
    }, delay);
  }, [connect]);

  // Main lifecycle: connect when we have a token, disconnect when we don't
  useEffect(() => {
    mountedRef.current = true;

    if (accessToken) {
      connect(accessToken);
    } else {
      intentionalCloseRef.current = true;
      closeSocket();
      clearTimers();
      setConnected(false);
    }

    return () => {
      mountedRef.current = false;
      intentionalCloseRef.current = true;
      closeSocket();
      clearTimers();
      setConnected(false);
    };
  }, [accessToken, connect, closeSocket, clearTimers, setConnected]);

  const joinRoom = useCallback((roomId: string) => {
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(
        JSON.stringify({ type: WS_MESSAGE_TYPES.JOIN_ROOM, room_id: roomId }),
      );
    }
  }, []);

  const leaveRoom = useCallback((roomId: string) => {
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(
        JSON.stringify({ type: WS_MESSAGE_TYPES.LEAVE_ROOM, room_id: roomId }),
      );
    }
  }, []);

  return { joinRoom, leaveRoom };
}
