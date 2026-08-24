package accountcenter

import (
	"unicode"
	"unicode/utf8"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Tier is display-only account packaging metadata. It never grants access.
type Tier string

const (
	TierStandard Tier = "standard"
	TierPro      Tier = "pro"
)

// ThemeMode is the durable, cross-device color-scheme preference.
type ThemeMode string

const (
	ThemeModeSystem ThemeMode = "system"
	ThemeModeLight  ThemeMode = "light"
	ThemeModeDark   ThemeMode = "dark"
)

// AvatarMetadata identifies one private object owned by the avatar subsystem.
type AvatarMetadata struct {
	ObjectKey   string
	ContentType string
	ETag        string
	SizeBytes   int64
}

func (a AvatarMetadata) Empty() bool {
	return a == (AvatarMetadata{})
}

func (a AvatarMetadata) validate() error {
	if a.Empty() {
		return nil
	}
	if a.ObjectKey == "" || a.ETag == "" || a.SizeBytes <= 0 {
		return status.Error(codes.InvalidArgument, "avatar object metadata is incomplete")
	}
	switch a.ContentType {
	case "image/jpeg", "image/png", "image/webp":
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported avatar content type %q", a.ContentType)
	}
}

// Profile is the durable public presentation state for one configured account.
type Profile struct {
	DisplayName string
	Tier        Tier
	Avatar      AvatarMetadata
	Revision    uint64
}

// Preferences contains private durable UI preferences for one account.
type Preferences struct {
	Theme    ThemeMode
	Revision uint64
}

const (
	ProfileRevisionConflictReason     = "ACCOUNT_PROFILE_REVISION_CONFLICT"
	PreferencesRevisionConflictReason = "ACCOUNT_PREFERENCES_REVISION_CONFLICT"
	ErrorDomain                       = "athena.account_center"
)

var (
	ErrProfileRevisionConflict     = stableError(codes.Aborted, ProfileRevisionConflictReason)
	ErrPreferencesRevisionConflict = stableError(codes.Aborted, PreferencesRevisionConflictReason)
)

func validateDisplayName(displayName string) error {
	if !utf8.ValidString(displayName) {
		return status.Error(codes.InvalidArgument, "display name must be valid UTF-8")
	}
	count := utf8.RuneCountInString(displayName)
	if count < 1 || count > 80 {
		return status.Error(codes.InvalidArgument, "display name must contain between 1 and 80 characters")
	}
	for _, r := range displayName {
		if unicode.IsControl(r) {
			return status.Error(codes.InvalidArgument, "display name must not contain control characters")
		}
	}
	return nil
}

func validateTier(tier Tier) error {
	switch tier {
	case TierStandard, TierPro:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported account tier %q", tier)
	}
}

func validateThemeMode(theme ThemeMode) error {
	switch theme {
	case ThemeModeSystem, ThemeModeLight, ThemeModeDark:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported account theme %q", theme)
	}
}

func validateProfile(profile Profile) error {
	if err := validateDisplayName(profile.DisplayName); err != nil {
		return err
	}
	if err := validateTier(profile.Tier); err != nil {
		return err
	}
	return profile.Avatar.validate()
}

func stableError(code codes.Code, reason string) error {
	errStatus := status.New(code, reason)
	withDetails, err := errStatus.WithDetails(&errdetails.ErrorInfo{
		Reason: reason,
		Domain: ErrorDomain,
	})
	if err != nil {
		return errStatus.Err()
	}
	return withDetails.Err()
}
