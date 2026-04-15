package installer

import (
	"encoding/json"
	"fmt"

	"github.com/rockship-co/clawkit/internal/dashboard"
	"github.com/rockship-co/clawkit/internal/ui"
)

func registryToJSON(reg *Registry) ([]byte, error) {
	return json.Marshal(reg)
}

// CmdDashboard starts the clawkit web dashboard.
func CmdDashboard(port int) {
	reg, err := loadRegistry()
	if err != nil {
		ui.Fatal("%v", err)
	}

	data, err := registryToJSON(reg)
	if err != nil {
		ui.Fatal("marshal registry: %v", err)
	}

	fmt.Printf("\n▸ Starting Clawkit Dashboard on port %d\n\n", port)
	if err := dashboard.Serve(port, data); err != nil {
		ui.Fatal("%v", err)
	}
}
