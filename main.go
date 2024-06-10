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

package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/v60/github"
	"github.com/stacklok/frizbee-action/pkg/action"
	"github.com/stacklok/frizbee/pkg/replacer"
	"github.com/stacklok/frizbee/pkg/utils/config"

	"golang.org/x/oauth2"
	"log"
	"os"
	"strings"
)

func main() {
	ctx := context.Background()
	// Initialize the frizbee action
	frizbeeAction, err := initAction(ctx)
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
func initAction(ctx context.Context) (*action.FrizbeeAction, error) {
	// Get the GitHub token from the environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}

	// Create a new GitHub client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	// Get the GITHUB_REPOSITORY_OWNER
	repoOwner := os.Getenv("GITHUB_REPOSITORY_OWNER")
	if repoOwner == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY_OWNER environment variable is not set")
	}

	// Split the GITHUB_REPOSITORY environment variable to get repo name
	repoFullName := os.Getenv("GITHUB_REPOSITORY")
	if repoFullName == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY environment variable is not set")
	}

	// Read the action settings from the environment and create the new frizbee replacers for actions and images
	return &action.FrizbeeAction{
		Client:            github.NewClient(tc),
		RepoOwner:         repoOwner,
		RepoName:          strings.TrimPrefix(repoFullName, repoOwner+"/"),
		ActionsPath:       os.Getenv("INPUT_ACTIONS"),
		DockerfilesPath:   os.Getenv("INPUT_DOCKERFILES"),
		KubernetesPath:    os.Getenv("INPUT_KUBERNETES"),
		DockerComposePath: os.Getenv("INPUT_DOCKER_COMPOSE"),
		OpenPR:            os.Getenv("INPUT_OPEN_PR") == "true",
		FailOnUnpinned:    os.Getenv("INPUT_FAIL_ON_UNPINNED") == "true",
		ActionsReplacer:   replacer.NewGitHubActionsReplacer(&config.Config{}).WithGitHubClientFromToken(token),
		ImagesReplacer:    replacer.NewContainerImagesReplacer(&config.Config{}),
	}, nil
}
