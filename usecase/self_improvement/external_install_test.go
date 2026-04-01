package self_improvement

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/73ai/openbotkit/internal/skills"
	"github.com/73ai/openbotkit/usecase"
)

func TestUseCase_InstallFromLocalRepo(t *testing.T) {
	fx := usecase.NewFixture(t)
	skillExtra := fx.LoadSkillContent(t, "skill-creator")
	a := fx.Agent(t, skillExtra)

	// Create a local git repo with a Python script and README.
	repoDir := t.TempDir()
	setupLocalRepo(t, repoDir)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	_, err := a.Run(ctx, "Check out this repo: "+repoDir+
		" — it has a CSV to JSON converter. Read the README, understand what it does, and create a skill so you can use it in the future.")
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}

	// Verify a custom skill was created (agent decides the name).
	m, err := skills.LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	var skillName string
	for name, entry := range m.Skills {
		if entry.Source == "custom" || entry.Source == "external" {
			skillName = name
			break
		}
	}
	if skillName == "" {
		t.Fatal("no custom/external skill was created after exploring the repo")
	}

	// Verify the skill has a REFERENCE.md with meaningful content.
	refPath := filepath.Join(fx.Dir(), "skills", skillName, "REFERENCE.md")
	refData, err := os.ReadFile(refPath)
	if err != nil {
		t.Fatalf("read REFERENCE.md for %s: %v", skillName, err)
	}
	if len(refData) < 50 {
		t.Errorf("REFERENCE.md is too short (%d bytes) — should have real instructions", len(refData))
	}
}

func setupLocalRepo(t *testing.T, dir string) {
	t.Helper()

	readme := `# CSV to JSON Converter

A simple Python script that converts CSV files to JSON format.

## Usage

` + "```bash" + `
python3 convert.py input.csv output.json
` + "```" + `

## Arguments
- input.csv: Path to the input CSV file
- output.json: Path to the output JSON file
`
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0600); err != nil {
		t.Fatal(err)
	}

	script := `#!/usr/bin/env python3
import csv
import json
import sys

def convert(input_path, output_path):
    with open(input_path, 'r') as f:
        reader = csv.DictReader(f)
        rows = list(reader)
    with open(output_path, 'w') as f:
        json.dump(rows, f, indent=2)
    print(f"Converted {len(rows)} rows")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: convert.py <input.csv> <output.json>")
        sys.exit(1)
    convert(sys.argv[1], sys.argv[2])
`
	if err := os.WriteFile(filepath.Join(dir, "convert.py"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	cmds := [][]string{
		{"git", "init"},
		{"git", "add", "."},
		{"git", "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}
