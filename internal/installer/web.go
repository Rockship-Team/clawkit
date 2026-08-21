package installer

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rockship-co/clawkit/internal/config"
	"github.com/rockship-co/clawkit/internal/ui"
)

// CmdWeb serves the static web entry point for a skill if it exists.
func CmdWeb(skillName string) {
	indexPath := findSkillWebIndex(skillName)
	if indexPath == "" {
		ui.Fatal("Skill '%s' does not have a web/ directory with index.html", skillName)
	}

	webDir := filepath.Dir(indexPath)
	absPath, err := filepath.Abs(webDir)
	if err != nil {
		ui.Fatal("Could not resolve web path: %v", err)
	}

	port := 7432
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      http.FileServer(http.Dir(absPath)),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	ui.Info("Serving web for %s", skillName)
	ui.Info("Local URL: http://localhost:%d", port)
	ui.Info("Web directory: %s", absPath)
	ui.Info("Press Ctrl+C to stop")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		ui.Fatal("Could not start web server: %v", err)
	}
}

func findSkillWebIndex(skillName string) string {
	skillDir := filepath.Join(config.GetSkillsDir(), skillName)
	indexPath := filepath.Join(skillDir, "web", "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		return indexPath
	}
	return ""
}
