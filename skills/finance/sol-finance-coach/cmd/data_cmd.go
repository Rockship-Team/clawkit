package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cmdData(args []string) {
	if len(args) == 0 {
		errOut("usage: data refresh")
		os.Exit(1)
	}

	switch args[0] {
	case "refresh":
		refreshData()
	default:
		errOut("unknown data command: " + args[0])
		os.Exit(1)
	}
}

func refreshData() {
	// Prefer prebuilt crawl binary in the skill directory.
	bins := []struct {
		path string
		dir  string
	}{
		{path: filepath.Join(skillDir(), "crawl"), dir: skillDir()},
		{path: filepath.Join("skills", "finance", "sol-finance-coach", "crawl"), dir: "."},
		{path: filepath.Join("..", "crawl"), dir: ".."},
	}
	errMsgs := make([]string, 0)
	for _, bin := range bins {
		if _, err := os.Stat(bin.path); err != nil {
			continue
		}
		cmd := exec.Command(bin.path, "all")
		cmd.Dir = bin.dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			errMsgs = append(errMsgs, "crawl("+bin.path+"): "+err.Error()+": "+string(out))
			continue
		}
		okOut(map[string]interface{}{"refreshed": true, "runner": "crawl", "output": string(out)})
		return
	}

	// Fallback to go run tools/crawl for development environments.
	devDirs := []string{
		filepath.Join(skillDir(), "tools", "crawl"),
		filepath.Join("skills", "finance", "sol-finance-coach", "tools", "crawl"),
		filepath.Join("..", "tools", "crawl"),
	}
	for _, devDir := range devDirs {
		if _, err := os.Stat(devDir); err != nil {
			continue
		}
		cmd := exec.Command("go", "run", ".", "all")
		cmd.Dir = devDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			errMsgs = append(errMsgs, "go run("+devDir+"): "+err.Error()+": "+string(out))
			continue
		}
		okOut(map[string]interface{}{"refreshed": true, "runner": "go run", "output": string(out)})
		return
	}

	// Last fallback for legacy installed layout without vertical grouping.
	legacyDir := filepath.Join("skills", "sol-finance-coach", "tools", "crawl")
	if _, err := os.Stat(legacyDir); err == nil {
		cmd := exec.Command("go", "run", ".", "all")
		cmd.Dir = legacyDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			errMsgs = append(errMsgs, "go run("+legacyDir+"): "+err.Error()+": "+string(out))
		} else {
			okOut(map[string]interface{}{"refreshed": true, "runner": "go run", "output": string(out)})
			return
		}
	}

	if len(errMsgs) > 0 {
		errOut("data refresh failed after trying all runners: " + strings.Join(errMsgs, " | "))
		os.Exit(1)
	}

	errOut("no crawl runner found (expected crawl binary or tools/crawl)")
	os.Exit(1)
}
