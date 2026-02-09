export const API_BASE = "/api/v1";

export const WS_MESSAGE_TYPES = {
  JOIN_ROOM: "join_room",
  LEAVE_ROOM: "leave_room",
  ROOM_PRESENCE: "room_presence",
  USER_JOINED: "user_joined",
  USER_LEFT: "user_left",
  NOTIFICATION: "notification",
} as const;

export const FEED_PAGE_SIZE = 20;
export const SUGGESTIONS_LIMIT = 5;

export const JWT_STORAGE_KEY = "pulse_access_token";
