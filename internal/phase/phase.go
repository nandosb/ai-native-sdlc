package phase

import (
	"github.com/yalochat/agentic-sdlc/internal/engine"
)

// RegisterAll registers all phase runners with the engine.
func RegisterAll(eng *engine.Engine) {
	eng.RegisterPhase(&Bootstrap{})
	eng.RegisterPhase(&Design{})
	eng.RegisterPhase(&Planning{})
	eng.RegisterPhase(&Tracking{})
	eng.RegisterPhase(&Executing{})
}
