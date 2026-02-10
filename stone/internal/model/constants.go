package model

// ValidReactionKinds is the single source of truth for allowed semantic reaction kinds.
// Any change here is automatically reflected in validation, seeding, and API layers.
var ValidReactionKinds = []string{
	"gave_me_energy",
	"calmed_me",
	"on_repeat",
	"surprised_me",
	"my_aesthetic",
}

// validReactionKindSet is a precomputed lookup set for O(1) validation.
var validReactionKindSet map[string]struct{}

func init() {
	validReactionKindSet = make(map[string]struct{}, len(ValidReactionKinds))
	for _, k := range ValidReactionKinds {
		validReactionKindSet[k] = struct{}{}
	}
}

// IsValidReactionKind reports whether the given kind is a recognized semantic reaction.
func IsValidReactionKind(kind string) bool {
	_, ok := validReactionKindSet[kind]
	return ok
}

// ValidContentTypes enumerates all supported content types.
var ValidContentTypes = []string{
	ContentTypeImage,
	ContentTypeVideo,
	ContentTypeShortVideo,
	ContentTypeText,
}

var validContentTypeSet map[string]struct{}

func init() {
	validContentTypeSet = make(map[string]struct{}, len(ValidContentTypes))
	for _, ct := range ValidContentTypes {
		validContentTypeSet[ct] = struct{}{}
	}
}

// IsValidContentType reports whether the given content type is recognized.
func IsValidContentType(contentType string) bool {
	_, ok := validContentTypeSet[contentType]
	return ok
}

// ValidEventTypes enumerates all recognized analytics event types.
var ValidEventTypes = []string{
	"view",
	"dwell",
	"skip",
	"replay",
	"save",
	"reaction",
	"path_follow",
	"path_followed",
	"follow_path",
	"tag_explore",
	"tag_open",
	"enter_room",
	"room_entered",
	"follow",
}

var validEventTypeSet map[string]struct{}

func init() {
	validEventTypeSet = make(map[string]struct{}, len(ValidEventTypes))
	for _, et := range ValidEventTypes {
		validEventTypeSet[et] = struct{}{}
	}
}

// IsValidEventType reports whether the given event type is recognized.
func IsValidEventType(eventType string) bool {
	_, ok := validEventTypeSet[eventType]
	return ok
}

// ValidTargetTypes enumerates all recognized event target types.
// Empty string is valid (events without a target entity).
var ValidTargetTypes = []string{
	"",
	"user",
	"content",
	"path",
	"room",
	"tag",
}

var validTargetTypeSet map[string]struct{}

func init() {
	validTargetTypeSet = make(map[string]struct{}, len(ValidTargetTypes))
	for _, tt := range ValidTargetTypes {
		validTargetTypeSet[tt] = struct{}{}
	}
}

// IsValidTargetType reports whether the given target type is recognized.
func IsValidTargetType(targetType string) bool {
	_, ok := validTargetTypeSet[targetType]
	return ok
}

// DefaultSignalWeights maps event types to their default affinity signal weight.
// Positive values increase affinity; negative values decrease it.
// The "view" and "dwell" types use dynamic calculation based on metadata,
// so their entry here represents a baseline reference only.
var DefaultSignalWeights = map[string]float64{
	"view":          0.0, // computed dynamically from dwell_ms
	"dwell":         0.0, // computed dynamically from dwell_ms
	"skip":          0.0, // computed dynamically from at_ms (negative)
	"replay":        1.0,
	"save":          1.5,
	"reaction":      1.2,
	"path_follow":   2.0,
	"path_followed": 2.0,
	"follow_path":   2.0,
	"tag_explore":   0.4,
	"tag_open":      0.4,
	"enter_room":    0.2,
	"room_entered":  0.2,
	"follow":        1.8,
}

// MediaContentTypes returns content types that require a media file (not text).
var MediaContentTypes = []string{
	ContentTypeImage,
	ContentTypeVideo,
	ContentTypeShortVideo,
}

// IsMediaContentType reports whether the content type requires a media file.
func IsMediaContentType(contentType string) bool {
	for _, ct := range MediaContentTypes {
		if ct == contentType {
			return true
		}
	}
	return false
}

// BridgeMinTagUsage is the minimum tag usage_count for a tag to appear in
// bridge explanations. Tags below this threshold are filtered to protect
// user privacy (very niche tags could reveal personal interests).
const BridgeMinTagUsage = 3
