package cli

import (
	"os/exec"
	"strings"
)

// gcloudAccounts returns logged-in gcloud accounts.
// Returns an empty slice (not error) if gcloud is not found.
func gcloudAccounts() ([]string, error) {
	out, err := exec.Command("gcloud", "auth", "list", "--format=value(account)").Output()
	if err != nil {
		if isExecNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return parseLines(string(out)), nil
}

// gcloudProjects returns GCP projects visible to the active account.
// Returns an empty slice (not error) if gcloud is not found.
func gcloudProjects() ([]string, error) {
	out, err := exec.Command("gcloud", "projects", "list", "--format=value(projectId)").Output()
	if err != nil {
		if isExecNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return parseLines(string(out)), nil
}

func parseLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func isExecNotFound(err error) bool {
	_, ok := err.(*exec.Error)
	return ok
}
