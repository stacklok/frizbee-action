//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pull_request

import (
	"log"
	"os"
	"os/exec"
)

func runCommand(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to run command %s %v: %v", name, args, err)
	}
}

func CommitAndPush() {
	// Configure git
	runCommand("git", "config", "--global", "--add", "safe.directory", "/github/workspace")
	runCommand("git", "config", "--global", "user.name", "frizbee-action[bot]")
	runCommand("git", "config", "--global", "user.email", "frizbee-action[bot]@users.noreply.github.com")

	// Get git status
	runCommand("git", "status")

	// Create a new branch
	branchName := "modify-workflows"
	runCommand("git", "checkout", "-b", branchName)

	// Add changes
	runCommand("git", "add", ".")

	// Commit changes
	runCommand("git", "commit", "-m", "frizbee: pin images and actions to commit hash")

	// Show the changes
	runCommand("git", "show")

	// Push changes
	runCommand("git", "push", "origin", branchName, "--force")
}

func CreatePullRequest() {
	title := "Frizbee: Pin images and actions to commit hash"
	body := "This PR pins images and actions to their commit hash"
	head := "modify-workflows"
	base := "main"
	runCommand("gh", "pr", "create", "--title", title, "--body", body, "--head", head, "--base", base)
}
