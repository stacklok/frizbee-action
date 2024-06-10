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

package action

import (
	"context"
	"fmt"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/google/go-github/v60/github"
	"github.com/stacklok/frizbee-action/pkg/pull_request"
	"github.com/stacklok/frizbee/pkg/replacer"
	"log"
	"os"
	"path/filepath"
)

type FrizbeeAction struct {
	Client            *github.Client
	RepoOwner         string
	RepoName          string
	ActionsPath       string
	DockerfilesPath   string
	KubernetesPath    string
	DockerComposePath string
	OpenPR            bool
	FailOnUnpinned    bool
	ActionsReplacer   *replacer.Replacer
	ImagesReplacer    *replacer.Replacer
}

// Run runs the frizbee action
func (fa *FrizbeeAction) Run(ctx context.Context) error {
	// Parse the workflow files
	modified, err := fa.parseWorkflowActions(ctx)
	if err != nil {
		return fmt.Errorf("failed to parse workflow files: %w", err)
	}

	// Parse all yaml/yml files referencing container images
	m, err := fa.parseImages(ctx)
	if err != nil {
		return fmt.Errorf("failed to parse image files: %w", err)
	}

	// Set the modified flag to true if any file was modified
	modified = modified || m

	// If the OpenPR flag is set, commit and push the changes and create a pull request
	if fa.OpenPR && modified {
		// TODO: use the git library to commit and push changes
		// TODO: perhaps refactor the code so instead of having 1 commit, we have separate commits for each file that
		// TODO: frizbee modified
		pull_request.CommitAndPush()
		// TODO: the default action token does not have permissions to open PRs against workflows in '.github/workflows/
		// TODO: We need to use a PAT or something else to fix this
		pull_request.CreatePullRequest()
	}

	// Exit with ErrUnpinnedFound error if any files were modified and the action is set to fail on unpinned
	if fa.FailOnUnpinned && modified {
		return ErrUnpinnedFound
	}

	return nil
}

// parseWorkflowActions parses the GitHub Actions workflow files and updates the modified files if the OpenPR flag is set
func (fa *FrizbeeAction) parseWorkflowActions(ctx context.Context) (bool, error) {
	if fa.ActionsPath == "" {
		log.Printf("Workflow path is empty")
		return false, nil
	}

	log.Printf("Parsing workflow files in %s...", fa.ActionsPath)
	res, err := fa.ActionsReplacer.ParsePath(ctx, fa.ActionsPath)
	if err != nil {
		return false, fmt.Errorf("failed to parse workflow files in %s: %w", fa.ActionsPath, err)
	}

	return fa.processOutput(res, fa.ActionsPath)
}

// parseImages parses the Dockerfiles, Docker Compose, and Kubernetes files for container images.
// It also updates the files if the OpenPR flag is set
func (fa *FrizbeeAction) parseImages(ctx context.Context) (bool, error) {
	var modified bool
	pathsToParse := []string{fa.DockerfilesPath, fa.DockerComposePath, fa.KubernetesPath}
	for _, path := range pathsToParse {
		if path == "" {
			continue
		}
		log.Printf("Parsing files for container images in %s", path)
		res, err := fa.ImagesReplacer.ParsePath(ctx, path)
		if err != nil {
			return false, fmt.Errorf("failed to parse: %w", err)
		}
		// Process the parsing output
		m, err := fa.processOutput(res, path)
		if err != nil {
			return false, fmt.Errorf("failed to process output: %w", err)
		}
		// Set the modified flag to true if any file was modified
		modified = modified || m
	}
	return modified, nil
}

// processOutput processes the output of a replacer, prints the processed and modified files and writes the
// changes to the files
func (fa *FrizbeeAction) processOutput(res *replacer.ReplaceResult, baseDir string) (bool, error) {
	var modified bool
	bfs := osfs.New(baseDir, osfs.WithBoundOS())

	// Show the processed files
	for _, path := range res.Processed {
		log.Printf("Processed file: %s", filepath.Base(path))
	}

	// Process the modified files
	for path, content := range res.Modified {
		log.Printf("Modified file: %s", filepath.Base(path))
		log.Printf("Modified content:\n%s\n", content)
		// Overwrite the content of the file with the changes if the OpenPR flag is set
		if fa.OpenPR {
			f, err := bfs.OpenFile(filepath.Base(path), os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return modified, fmt.Errorf("failed to open file %s: %w", filepath.Base(path), err)
			}
			defer func() {
				if err := f.Close(); err != nil {
					log.Fatalf("failed to close file %s: %v", filepath.Base(path), err) // nolint:errcheck
				}
			}()
			_, err = fmt.Fprintf(f, "%s", content)
			if err != nil {
				return modified, fmt.Errorf("failed to write to file %s: %w", filepath.Base(path), err)
			}
			// Set the modified flag to true if any file was modified
			modified = true
		}
	}
	return modified, nil
}
