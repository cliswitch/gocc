package config

import "strings"

const (
	LevelHaiku    = "haiku"
	LevelSonnet   = "sonnet"
	LevelOpus     = "opus"
	LevelSubagent = "subagent"
)

var levelPrefixes = map[string]string{
	LevelHaiku:    "h",
	LevelSonnet:   "s",
	LevelOpus:     "o",
	LevelSubagent: "sa",
}

var prefixToLevel = map[string]string{
	"h":  LevelHaiku,
	"s":  LevelSonnet,
	"o":  LevelOpus,
	"sa": LevelSubagent,
}

func AnnotateModel(level, model string) string {
	prefix, ok := levelPrefixes[level]
	if !ok {
		return model
	}
	return prefix + "-" + model
}

func ParseAnnotatedModel(annotated string) (level, model string, ok bool) {
	idx := strings.Index(annotated, "-")
	if idx < 0 {
		return "", annotated, false
	}
	prefix := annotated[:idx]
	lvl, known := prefixToLevel[prefix]
	if !known {
		return "", annotated, false
	}
	return lvl, annotated[idx+1:], true
}

func (m Models) ModelForLevel(level string) string {
	switch level {
	case LevelHaiku:
		return m.HaikuModel
	case LevelSonnet:
		return m.SonnetModel
	case LevelOpus:
		return m.OpusModel
	case LevelSubagent:
		return m.SubagentModel
	default:
		return ""
	}
}

// LevelForModel returns the level for a given model name by reverse lookup.
// Returns empty string if the model is not found in any level.
func (m Models) LevelForModel(model string) string {
	if model == "" {
		return ""
	}
	if m.HaikuModel == model {
		return LevelHaiku
	}
	if m.SonnetModel == model {
		return LevelSonnet
	}
	if m.OpusModel == model {
		return LevelOpus
	}
	if m.SubagentModel == model {
		return LevelSubagent
	}
	return ""
}

// NeedAnnotation returns true if any two non-empty level model names collide,
// meaning annotation is required to distinguish them.
func (m Models) NeedAnnotation() bool {
	models := [...]string{m.HaikuModel, m.SonnetModel, m.OpusModel, m.SubagentModel}
	for i := 0; i < len(models); i++ {
		if models[i] == "" {
			continue
		}
		for j := i + 1; j < len(models); j++ {
			if models[j] != "" && models[i] == models[j] {
				return true
			}
		}
	}
	return false
}
