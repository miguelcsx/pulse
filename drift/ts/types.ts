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
  is_following: boolean;
  is_blocked: boolean;
  trust_profile?: TrustProfile;
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

// Feed content moment with optional room context
export interface FeedMoment extends Content {
  room_context?: RoomContext;
}

export interface AffinityFeedItem {
  id: string;
  unit_type: "moment" | "ask";
  content?: FeedMoment;
  bridge?: Bridge;
  room_context?: RoomContext;
  reason?: string;
  created_at: string;
}

// Feed
export interface FeedResponse {
  items: AffinityFeedItem[];
  next_cursor: string;
  has_more: boolean;
}

export interface ContentFeedResponse {
  items: Content[];
  next_cursor: string;
  has_more: boolean;
}

// Human advice network
export type DesiredHelpType = "advice" | "peer" | "mentor" | "feedback";
export type AskUrgency = "now" | "soon" | "this_week" | "exploring";
export type AskVisibility = "private" | "community" | "public";
export type BridgeType = "mentor" | "peer" | "adjacent_perspective";
export type BridgeStatus = "suggested" | "asked" | "responded" | "dismissed";
export type HelpSignalKind =
  | "useful"
  | "clarifying"
  | "motivating"
  | "practical"
  | "not_relevant";
export type Availability = "async" | "live_now" | "bookable_10m";

export interface Ask {
  id: string;
  user_id: string;
  user?: User;
  question: string;
  triage_summary: string;
  topic: string;
  urgency: AskUrgency;
  desired_help_type: DesiredHelpType;
  visibility: AskVisibility;
  anonymous: boolean;
  created_at: string;
  updated_at: string;
}

export interface AskVisibilityInput {
  visibility: AskVisibility;
  anonymous: boolean;
}

// CommonsEntry — an answered ask published to the Commons. When the ask is
// anonymous the backend strips the asker identity (user_id is the zero UUID).
export interface CommonsEntry {
  ask: Ask;
  responses: BridgeResponse[];
}

// NetworkConnection — a person you've actually exchanged perspective with.
export interface NetworkConnection {
  user: User;
  direction: "you_asked" | "you_answered";
  topic: string;
  question: string;
  last_at: string;
}

export interface AskCreateInput {
  question: string;
  topic?: string;
  urgency?: AskUrgency;
  desired_help_type?: DesiredHelpType;
  visibility?: AskVisibility;
}

export interface Bridge {
  id: string;
  ask_id: string;
  ask?: Ask;
  requester_id: string;
  recommended_user_id: string;
  recommended_user: User;
  reason: string;
  bridge_type: BridgeType;
  confidence: number;
  status: BridgeStatus;
  responses?: BridgeResponse[];
  created_at: string;
  updated_at: string;
}

export interface BridgeResponse {
  id: string;
  bridge_id: string;
  responder_id: string;
  responder?: User;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface AskCreateResponse {
  ask: Ask;
  bridges: Bridge[];
}

export interface HelpSignal {
  id: string;
  bridge_id: string;
  user_id: string;
  kind: HelpSignalKind;
  created_at: string;
}

export interface TrustProfile {
  user_id: string;
  user?: User;
  topics: string;
  lived_experience: string;
  availability: Availability;
  helped_count: number;
  response_quality: number;
  created_at: string;
  updated_at: string;
}

export interface TrustProfileInput {
  topics: string;
  lived_experience: string;
  availability: Availability;
}

export interface HelpSession {
  id: string;
  title: string;
  intent: string;
  description: string;
  member_count: number;
  expires_at: string;
  created_at: string;
}

export interface TodayResponse {
  latest_ask?: Ask;
  bridges: Bridge[];
  incoming_bridges: Bridge[];
  help_sessions: HelpSession[];
  trust_profile?: TrustProfile;
  starter_prompts: string[];
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
