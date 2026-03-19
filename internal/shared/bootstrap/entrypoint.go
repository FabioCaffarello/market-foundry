package bootstrap

import (
	"flag"
	"fmt"
	"os"

	"internal/shared/settings"
)

// Main is the canonical entrypoint for all foundry binaries.
// It handles flag parsing, config loading/validation and error reporting,
// then delegates to the service-specific run function.
func Main(serviceName string, run func(settings.AppConfig)) {
	configPath := flag.String("config", "config.jsonc", "path to JSONC config file")
	flag.Parse()

	cfg, prob := LoadAndValidate(*configPath)
	if prob != nil {
		fmt.Fprintf(os.Stderr, "%s: config error: %v\n", serviceName, prob)
		os.Exit(1)
	}

	run(cfg)
}
