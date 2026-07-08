package provisioning

import _ "embed"

const defaultEmbeddedSourceName = "embedded:drun-defaults"

//go:embed defaults/provisionings.yaml
var defaultProvisioningsManifest []byte

// DefaultEmbeddedSources returns the built-in provisioning catalogs shipped
// with drun as the final runtime fallback.
func DefaultEmbeddedSources() []EmbeddedSource {
	return []EmbeddedSource{{
		Name:    defaultEmbeddedSourceName,
		Content: append([]byte(nil), defaultProvisioningsManifest...),
	}}
}
