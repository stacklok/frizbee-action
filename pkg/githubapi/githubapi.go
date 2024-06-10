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

package githubapi

import (
	"context"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

type GitHubClient struct {
	Client *github.Client
	Ctx    context.Context
}

// NewGitHubClient creates a new instance of GitHubClient with the provided token.
// It initializes the GitHub API client using the token for authentication.
// The returned GitHubClient can be used to interact with the GitHub API.
func NewGitHubClient(token string) *GitHubClient {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &GitHubClient{
		Client: client,
		Ctx:    ctx,
	}
}
