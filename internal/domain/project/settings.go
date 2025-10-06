package project

// SettingsManager manages project settings
type SettingsManager struct {
	project *Project
}

// NewSettingsManager creates a new settings manager
func NewSettingsManager(project *Project) *SettingsManager {
	return &SettingsManager{
		project: project,
	}
}

// Get retrieves a setting value
func (sm *SettingsManager) Get(key string) (string, bool) {
	return sm.project.GetSetting(key)
}

// GetWithDefault retrieves a setting value or returns a default
func (sm *SettingsManager) GetWithDefault(key, defaultValue string) string {
	if value, exists := sm.project.GetSetting(key); exists {
		return value
	}
	return defaultValue
}

// Has checks if a setting exists
func (sm *SettingsManager) Has(key string) bool {
	_, exists := sm.project.GetSetting(key)
	return exists
}

// All returns all settings
func (sm *SettingsManager) All() map[string]string {
	// Return a copy to prevent external modification
	settings := make(map[string]string, len(sm.project.Settings))
	for k, v := range sm.project.Settings {
		settings[k] = v
	}
	return settings
}
