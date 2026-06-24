package cli

import (
	"fmt"

	"github.com/aleksandarknezevic/chainform/internal/config"
	"github.com/aleksandarknezevic/chainform/internal/resource"
)

// ValidateConfig runs schema-level and provider-level validation without
// contacting the chain.
func ValidateConfig(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	for _, rc := range cfg.Resources {
		if _, err := resource.Build(rc); err != nil {
			return fmt.Errorf("resource %q %q: %w", rc.Type, rc.Name, err)
		}
	}
	return nil
}
