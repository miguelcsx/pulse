// User
export interface User {
  id: string;
  handle: string;
  email?: string;
  display_name: string;
  bio: string;
  avatar_url: string;
  location: string;
  created_at: string;
  updated_at: string;
}

export interface UserProfile extends User {
  follower_count: number;
  following_count: number;
  content_count: number;
  is_following?: boolean;
  is_blocked?: boolean;
}

// Auth
export interface AuthTokens {
  access_token: string;
  refresh_token?: string;
  user: User;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  handle: string;
  email: string;
  password: string;
  display_name: string;
}

// Content — supports image, video, short_video, text
export type ContentType = "image" | "video" | "short_video" | "text";

export interface Content {
  id: string;
  creator_id: string;
  creator: User;
  content_type: ContentType;
  media_asset_id?: string;
  media_url: string;
  body: string; // text content or caption
  tags: Tag[];
  reactions?: Record<string, number>; // kind -> count
  created_at: string;
}

export interface ContentUpload {
  content_type: ContentType;
  body: string;
  tags: string[]; // user-defined hashtag strings
  media_asset_id?: string;
}

export type MediaAssetStatus =
  | "initiated"
  | "uploaded"
  | "processing"
  | "ready"
  | "failed";

export interface MediaAsset {
  id: string;
  owner_id: string;
  content_type: ContentType;
  original_path: string;
  playback_path: string;
  filename: string;
  mime_type: string;
  size_bytes: number;
  status: MediaAssetStatus;
  error_message: string;
  ready_at?: string;
  original_url?: string;
  playback_url?: string;
  created_at: string;
  updated_at: string;
}

// Tags — user-defined hashtags, created on first use
export interface Tag {
  id: string;
  name: string;
  usage_count: number;
  created_at: string;
}

// Semantic Reactions (replaces likes)
export type ReactionKind =
  | "gave_me_energy"
  | "calmed_me"
  | "on_repeat"
  | "surprised_me"
  | "my_aesthetic";

export const REACTION_LABELS: Record<ReactionKind, string> = {
  gave_me_energy: "Gave me energy",
  calmed_me: "Calmed me",
  on_repeat: "On repeat",
  surprised_me: "Surprised me",
  my_aesthetic: "My aesthetic",
};

// Room context for feed items
export interface RoomContext {
  room_id: string;
  tags: string[];
  member_count: number;
}

// Feed item with optional room context
export interface FeedItem extends Content {
  room_context?: RoomContext;
}

// Feed
export interface FeedResponse {
  items: FeedItem[];
  next_cursor: string;
  has_more: boolean;
  suggestions?: Suggestion[];
}

// Suggestion types — buckets for discovery
export type SuggestionType =
  | "closest_twin"    // behavioral affinity (dwell/reactions)
  | "adjacent_taste"  // tag overlap
  | "path_affinity"   // followed 2+ paths from same creator
  | "serendipity";    // different profile, 1 strong bridge

// Social — suggestions with Bridges (why you connect)
export interface Suggestion {
  user: User;
  shared_tags: number;
  common_tags: Tag[];
  bridge: string; // human-readable explanation of why you connect
  affinity_score?: number;
  suggestion_type?: SuggestionType;
  path_count?: number;
}

// Discover — aggregated endpoint
export interface DiscoverResponse {
  suggestions: Suggestion[];
  closest_twins?: Suggestion[];
  adjacent_taste?: Suggestion[];
  serendipity?: Suggestion[];
  rooms: Room[];
  paths: Path[];
}

// Rooms — mood rooms for co-consumption
export interface Room {
  id: string;
  cluster_key: string;
  tags: Tag[];
  member_count: number;
  expires_at: string;
  created_at: string;
}

export interface RoomMember {
  user_id: string;
  user: User;
  joined_at: string;
}

// Paths — curated content journeys (can include any content type)
export interface Path {
  id: string;
  creator_id: string;
  creator: User;
  title: string;
  description: string;
  system_generated?: boolean;
  items: PathItem[];
  follower_count: number;
  is_following?: boolean;
  created_at: string;
}

export interface PathItem {
  id: string;
  content_id: string;
  content: Content;
  position: number;
  note: string;
}

// Events — user activity for affinity graph
export interface AppEvent {
  type: string; // view, dwell, skip, replay, save, path_follow, reaction, tag_explore
  target_type: string;
  target_id?: string;
  metadata?: Record<string, unknown>;
  timestamp: string;
}

// WebSocket
export interface WSMessage {
  type: string;
  room_id?: string;
  user_id?: string;
  member_count?: number;
  members?: User[];
  timestamp?: string;
  data?: unknown;
}

// API Response
export interface APIResponse<T> {
  data: T;
  error?: string;
}

// Pagination
export interface PaginatedResponse<T> {
  items: T[];
  next_cursor: string;
  has_more: boolean;
}
