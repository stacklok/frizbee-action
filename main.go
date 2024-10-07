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

// Package main provides the entrypoint for the frizbee action
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/go-github/v60/github"
	"github.com/stacklok/frizbee/pkg/replacer"
	"github.com/stacklok/frizbee/pkg/utils/config"

	"github.com/stacklok/frizbee-action/pkg/action"
)

func main() {
	ctx := context.Background()
	// Initialize the frizbee action
	frizbeeAction, err := initAction()
	if err != nil {
		log.Fatalf("Error initializing action: %v", err)
	}

	// Run the frizbee action
	err = frizbeeAction.Run(ctx)
	if err != nil {
		if errors.Is(err, action.ErrUnpinnedFound) {
			log.Printf("Unpinned actions or container images found. Check the Frizbee Action logs for more information.")
			os.Exit(1)
		}
		log.Fatalf("Error running action: %v", err)
	}
}

// initAction initializes the frizbee action - reads the environment variables, creates the GitHub client, etc.
func initAction() (*action.FrizbeeAction, error) {
	var repo *git.Repository
	var fs billy.Filesystem
	var githubClient *github.Client

	// Get the GitHub token from the environment
	token := os.Getenv("GITHUB_TOKEN")

	// Get the GITHUB_REPOSITORY_OWNER
	repoOwner := os.Getenv("GITHUB_REPOSITORY_OWNER")
	if repoOwner == "" {
		return nil, errors.New("GITHUB_REPOSITORY_OWNER environment variable is not set")
	}

	// Split the GITHUB_REPOSITORY environment variable to get repo name
	repoFullName := os.Getenv("GITHUB_REPOSITORY")
	if repoFullName == "" {
		return nil, errors.New("GITHUB_REPOSITORY environment variable is not set")
	}

	repoRoot := os.Getenv("INPUT_REPO_ROOT")
	if repoRoot == "" {
		if token == "" {
			return nil, errors.New("GITHUB_TOKEN environment variable is not set")
		}

		// Create a new GitHub client
		githubClient = github.NewClient(nil).WithAuthToken(token)

		// Clone the repository
		var err error
		fs, repo, err = cloneRepository("https://github.com/"+repoFullName, repoOwner, token)
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository: %w", err)
		}
	} else {
		fs = osfs.New(repoRoot)
		var err error
		repo, err = git.PlainOpen(repoRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to open repository: %w", err)
		}
	}

	cfg := config.DefaultConfig()
	excludeActions := os.Getenv("INPUT_ACTIONS_EXCLUDE")
	if excludeActions != "" {
		cfg.GHActions.Exclude = valToStrings(excludeActions)
	}
	excludeBranches := os.Getenv("INPUT_ACTIONS_EXCLUDE_BRANCHES")
	if excludeBranches != "" {
		cfg.GHActions.ExcludeBranches = valToStrings(excludeBranches)
	}
	excludeImages := os.Getenv("INPUT_IMAGES_EXCLUDE")
	if excludeImages != "" {
		cfg.Images.ExcludeImages = valToStrings(excludeImages)
	}
	excludeTags := os.Getenv("INPUT_IMAGES_EXCLUDE_TAGS")
	if excludeTags != "" {
		cfg.Images.ExcludeTags = valToStrings(excludeTags)
	}

	actionsPathList, err := actionsPathList()
	if err != nil {
		return nil, err
	}

	// Read the action settings from the environment and create the new frizbee replacers for actions and images
	return &action.FrizbeeAction{
		Client:    githubClient,
		Token:     token,
		RepoOwner: repoOwner,
		RepoName:  strings.TrimPrefix(repoFullName, repoOwner+"/"),

		ActionsPaths:       actionsPathList,
		DockerfilesPaths:   envToStrings("INPUT_DOCKERFILES"),
		KubernetesPaths:    envToStrings("INPUT_KUBERNETES"),
		DockerComposePaths: envToStrings("INPUT_DOCKER_COMPOSE"),

		OpenPR:          os.Getenv("INPUT_OPEN_PR") == "true",
		FailOnUnpinned:  os.Getenv("INPUT_FAIL_ON_UNPINNED") == "true",
		ActionsReplacer: replacer.NewGitHubActionsReplacer(cfg).WithGitHubClientFromToken(token),
		ImagesReplacer:  replacer.NewContainerImagesReplacer(cfg),
		BFS:             fs,
		Repo:            repo,
	}, nil
}

// cloneRepository clones the repository and returns a billy.Filesystem interface to interact with it
func cloneRepository(url, owner, accessToken string) (billy.Filesystem, *git.Repository, error) {
	fs := memfs.New()
	// Use memory storage to clone the repository in memory
	store := memory.NewStorage()
	repo, err := git.Clone(store, fs, &git.CloneOptions{
		URL: url,
		Auth: &http.BasicAuth{
			Username: owner,
			Password: accessToken,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	return fs, repo, nil
}

func envToStrings(env string) []string {
	return valToStrings(os.Getenv(env))
}

func valToStrings(val string) []string {
	var vals []string

	if val == "" {
		return []string{}
	}

	if err := json.Unmarshal([]byte(val), &vals); err != nil {
		log.Printf("Error unmarshalling %s: %v", val, err)
		return []string{}
	}

	return vals
}

func actionsPathList() ([]string, error) {
	actions := os.Getenv("INPUT_ACTIONS")
	actionsPaths := os.Getenv("INPUT_ACTIONS_PATHS")
	if actions != "" && actionsPaths != "" {
		return nil, errors.New("cannot set both INPUT_ACTIONS and INPUT_ACTIONS_PATHS")
	} else if actions == "" && actionsPaths == "" {
		// Default for actions was `.github/workflows``
		actions = ".github/workflows"
	}

	if actions != "" {
		return []string{actions}, nil
	}
	return valToStrings(actionsPaths), nil
}
