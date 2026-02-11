package phase

import "os"

const artifactsDir = ".sdlc/artifacts"

func ensureArtifactsDir() error {
	return os.MkdirAll(artifactsDir, 0755)
}
