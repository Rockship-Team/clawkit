package main

import (
	"github.com/rockship-co/clawkit/oauth"
)

func init() {
	// Wire the promptInput function from ui.go into the oauth package
	oauth.PromptInput = promptInput
}

// runOAuthFlow looks up the provider by name and runs its OAuth flow.
// Tokens are saved to the skill's config.
func runOAuthFlow(providerName, skillDir string) error {
	provider, err := oauth.Get(providerName)
	if err != nil {
		return err
	}

	tokens, err := provider.Authenticate()
	if err != nil {
		return err
	}

	// Save tokens to skill config
	cfg, _ := loadSkillConfig(skillDir)
	if cfg == nil {
		cfg = &SkillConfig{Tokens: map[string]string{}}
	}
	if cfg.Tokens == nil {
		cfg.Tokens = map[string]string{}
	}

	for k, v := range tokens {
		cfg.Tokens[k] = v
	}
	cfg.OAuthDone = true

	return saveSkillConfig(skillDir, cfg)
}
