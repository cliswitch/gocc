package config

import "strconv"

func (p *Profile) EnvVars() []string {
	var envs []string
	add := func(key, value string) {
		if value != "" {
			envs = append(envs, key+"="+value)
		}
	}

	if p.Protocol != ProtocolNative {
		add("ANTHROPIC_MODEL", p.Models.MainModel)
		annotate := p.Models.NeedAnnotation()
		modelValue := func(level, model string) string {
			if annotate {
				return AnnotateModel(level, model)
			}
			return model
		}
		if p.Models.HaikuModel != "" {
			add("ANTHROPIC_DEFAULT_HAIKU_MODEL", modelValue(LevelHaiku, p.Models.HaikuModel))
		}
		if p.Models.SonnetModel != "" {
			add("ANTHROPIC_DEFAULT_SONNET_MODEL", modelValue(LevelSonnet, p.Models.SonnetModel))
		}
		if p.Models.OpusModel != "" {
			add("ANTHROPIC_DEFAULT_OPUS_MODEL", modelValue(LevelOpus, p.Models.OpusModel))
		}
		if p.Models.SubagentModel != "" {
			add("CLAUDE_CODE_SUBAGENT_MODEL", modelValue(LevelSubagent, p.Models.SubagentModel))
		}
		if p.Models.SmallFastModel != "" {
			add("ANTHROPIC_SMALL_FAST_MODEL", p.Models.SmallFastModel)
		}

		if p.Reasoning.MaxOutputTokens > 0 {
			add("CLAUDE_CODE_MAX_OUTPUT_TOKENS", strconv.Itoa(p.Reasoning.MaxOutputTokens))
		}
		if p.Reasoning.MaxThinkingTokens > 0 {
			add("MAX_THINKING_TOKENS", strconv.Itoa(p.Reasoning.MaxThinkingTokens))
		}
		add("CLAUDE_CODE_EFFORT_LEVEL", p.Reasoning.EffortLevel)
	}

	add("HTTP_PROXY", p.Proxy.HTTPProxy)
	add("HTTPS_PROXY", p.Proxy.HTTPSProxy)
	add("NO_PROXY", p.Proxy.NoProxy)

	return envs
}
