package engine

// Artifact key constants — single source of truth for artifact names used
// across Go code, the web UI, and skill prompts.
const (
	ArtifactScopingDoc       = "scoping_doc"
	ArtifactScopingDocNotion = "scoping_doc_notion"
	ArtifactPERT             = "pert"
)

// ArtifactDef describes a pipeline artifact's metadata.
type ArtifactDef struct {
	Key         string `json:"key"`
	DefaultFile string `json:"default_file"`
	NotionTitle string `json:"notion_title"`
}

// Artifacts maps each artifact key to its definition.
var Artifacts = map[string]ArtifactDef{
	ArtifactScopingDoc: {Key: ArtifactScopingDoc, DefaultFile: "scoping-doc.md", NotionTitle: "1. Scoping Document"},
	ArtifactPERT:       {Key: ArtifactPERT, DefaultFile: "pert.md", NotionTitle: "2. PERT"},
}

// PhaseArtifact maps pipeline phases to their output artifact key.
var PhaseArtifact = map[Phase]string{
	PhaseDesign:   ArtifactScopingDoc,
	PhasePlanning: ArtifactPERT,
}
