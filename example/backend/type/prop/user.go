package prop

// UserMetadata represents user-specific metadata
// polygon sql/scanner
// polygon sql/override column:users.metadata
type UserMetadata struct {
	Preferences map[string]any `json:"preferences,omitempty"`
}

// UserSettings represents user-specific settings and preferences
// polygon sql/scanner
// polygon sql/override column:users.settings
type UserSettings struct {
	Theme *string `json:"theme,omitempty"`
}
