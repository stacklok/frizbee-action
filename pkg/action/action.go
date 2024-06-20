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

// Package action provides the actual frizbee action
package action

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v60/github"
	"github.com/stacklok/frizbee/pkg/replacer"
)

// FrizbeeAction is the main struct for the frizbee action
type FrizbeeAction struct {
	Client    *github.Client
	Token     string
	RepoOwner string
	RepoName  string

	ActionsPath        string
	DockerfilesPaths   []string
	KubernetesPaths    []string
	DockerComposePaths []string

	OpenPR          bool
	FailOnUnpinned  bool
	ActionsReplacer *replacer.Replacer
	ImagesReplacer  *replacer.Replacer
	BFS             billy.Filesystem
	Repo            *git.Repository
	bodyBuilder     *strings.Builder
}

// Run runs the frizbee action
func (fa *FrizbeeAction) Run(ctx context.Context) error {
	// result holds all the processed and modified files
	out := &replacer.ReplaceResult{Processed: make([]string, 0), Modified: make(map[string]string)}

	// Parse the workflow files
	if err := fa.parseWorkflowActions(ctx, out); err != nil {
		return fmt.Errorf("failed to parse workflow files: %w", err)
	}

	// Parse all yaml/yml files referencing container images
	if err := fa.parseImages(ctx, out); err != nil {
		return fmt.Errorf("failed to parse image files: %w", err)
	}
	log.Printf("Processing output...")
	// Process the output
	return fa.processOutput(ctx, out)
}

// parseWorkflowActions parses the GitHub Actions workflow files
func (fa *FrizbeeAction) parseWorkflowActions(ctx context.Context, out *replacer.ReplaceResult) error {
	if fa.ActionsPath == "" {
		log.Printf("Workflow path is empty")
		return nil
	}

	log.Printf("Parsing workflow files in %s...", fa.ActionsPath)
	res, err := fa.ActionsReplacer.ParsePathInFS(ctx, fa.BFS, fa.ActionsPath)
	if err != nil {
		return fmt.Errorf("failed to parse workflow files in %s: %w", fa.ActionsPath, err)
	}

	// Copy the processed and modified files to the output
	out.Processed = mapset.NewSet(out.Processed...).Union(mapset.NewSet(res.Processed...)).ToSlice()
	for key, value := range res.Modified {
		out.Modified[key] = value
	}
	return nil
}

// parseImages parses the Dockerfiles, Docker Compose, and Kubernetes files for container images.
func (fa *FrizbeeAction) parseImages(ctx context.Context, out *replacer.ReplaceResult) error {
	pathsToParse := []string{}
	pathsToParse = append(pathsToParse, fa.DockerfilesPaths...)
	pathsToParse = append(pathsToParse, fa.DockerComposePaths...)
	pathsToParse = append(pathsToParse, fa.KubernetesPaths...)

	for _, path := range pathsToParse {
		if path == "" {
			continue
		}
		log.Printf("Parsing files for container images in %s", path)
		res, err := fa.ImagesReplacer.ParsePathInFS(ctx, fa.BFS, path)
		if err != nil {
			return fmt.Errorf("failed to parse: %w", err)
		}
		// Copy the processed and modified files to the output
		out.Processed = mapset.NewSet(out.Processed...).Union(mapset.NewSet(res.Processed...)).ToSlice()
		for key, value := range res.Modified {
			out.Modified[key] = value
		}
	}
	return nil
}

// processOutput processes the output of a replacer, prints the processed and modified files and writes the
// changes to the files
func (fa *FrizbeeAction) processOutput(ctx context.Context, res *replacer.ReplaceResult) error {
	// Show the processed files
	if len(res.Processed) != 0 {
		log.Printf("Processed the following files:")
		for _, path := range res.Processed {
			log.Printf("* %s", path)
		}
	} else {
		log.Printf("No files were processed")
		return nil
	}

	if len(res.Modified) != 0 {
		log.Printf("Modified the following files:")
		// Process the modified files
		for path, content := range res.Modified {
			log.Printf("* %s", path)
			log.Printf("%s\n", content)
			// Overwrite the content of the file with the changes if the OpenPR flag is set
			if fa.OpenPR {
				if err := fa.commitChanges(path, content); err != nil {
					return fmt.Errorf("failed to commit changes: %w", err)
				}
			}
		}
		if fa.OpenPR {
			// Create a new pull request
			if err := fa.createPR(ctx); err != nil {
				return fmt.Errorf("failed to create PR: %w", err)
			}
		}
		// Fail if the FailOnUnpinned flag is set and any files were modified
		if fa.FailOnUnpinned {
			return ErrUnpinnedFound
		}
	}
	return nil
}

func (fa *FrizbeeAction) commitChanges(path, content string) error {
	f, err := fa.BFS.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalf("failed to close file %s: %v", path, err) // nolint:errcheck
		}
	}()
	_, err = fmt.Fprintf(f, "%s", content)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", path, err)
	}

	// Stage the file
	worktree, err := fa.Repo.Worktree()
	if err != nil {
		log.Fatalf("failed to get worktree: %v", err)
	}
	_, err = worktree.Add(path)
	if err != nil {
		log.Fatalf("failed to add file to staging area: %v", err)
	}

	// Commit the change
	_, err = worktree.Commit(fmt.Sprintf("Update %s by pinning its image references", path), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "github-actions[bot]",
			Email: "github-actions[bot]@users.noreply.github.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		log.Fatalf("failed to commit changes: %v", err)
	}
	return nil
}

// createPR creates a new pull request with the changes made so far
func (fa *FrizbeeAction) createPR(ctx context.Context) error {
	// Create a new branch for the PR
	branchName := "frizbee-action-patch"

	headRef, err := fa.Repo.Head()
	if err != nil {
		log.Fatalf("failed to get head reference: %v", err)
	}
	branchRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branchName), headRef.Hash())
	err = fa.Repo.Storer.SetReference(branchRef)
	if err != nil {
		log.Fatalf("failed to create branch: %v", err)
	}

	// Push the new branch
	err = fa.Repo.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: fa.RepoOwner,
			Password: fa.Token,
		},
		Force: true,
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName)),
		},
	})
	if err != nil {
		log.Fatalf("failed to push branch: %v", err)
	}

	log.Printf("Branch %s pushed successfully\n", branchName)

	// Check if a PR already exists for the branch and return if it does
	openPrs, _, err := fa.Client.PullRequests.List(ctx, fa.RepoOwner, fa.RepoName, &github.PullRequestListOptions{
		Head:  branchName,
		State: "open",
	})
	if err != nil {
		log.Fatalf("failed to list PRs: %v", err)
	}

	for _, pr := range openPrs {
		if pr.GetHead().GetRef() == branchName {
			log.Printf("PR %d already exists\n", pr.GetNumber())
			return nil
		}
	}

	// Get defaultBranch
	repository, _, err := fa.Client.Repositories.Get(ctx, fa.RepoOwner, fa.RepoName)
	if err != nil {
		log.Fatalf("Error getting repository details: %v", err)
	}
	defaultBranch := repository.GetDefaultBranch()

	fa.bodyBuilder = &strings.Builder{}
	fa.bodyBuilder.WriteString("## Frizbee: Pin images and actions to commit hash\n\n")
	fa.bodyBuilder.WriteString("The following PR pins images and actions to their commit hash.\n\n")
	fa.bodyBuilder.WriteString("Pinning images and actions to their commit hash ensures that the same " +
		"version of the image or action is used every time the workflow runs. This is important for " +
		"reproducibility and security.\n\n")
	//nolint:lll
	fa.bodyBuilder.WriteString("Pinning is a [security practice recommended by GitHub](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#using-third-party-actions).\n\n")
	//nolint:lll
	fa.bodyBuilder.WriteString("🥏 Posted on behalf of 🥏 [frizbee-action](https://github.com/stacklok/frizbee-action), by [Stacklok](https://stacklok.com).\n\n")

	// Create a new PR
	pr, _, err := fa.Client.PullRequests.Create(ctx, fa.RepoOwner, fa.RepoName, &github.NewPullRequest{
		Title:               github.String("Frizbee: Pin images and actions to commit hash"),
		Body:                github.String(fa.bodyBuilder.String()),
		Head:                github.String(branchName),
		Base:                github.String(defaultBranch),
		MaintainerCanModify: github.Bool(true),
	})
	if err != nil {
		return err
	}
	log.Printf("PR %d created successfully\n", pr.GetNumber())
	return nil
}
