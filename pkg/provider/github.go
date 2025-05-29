// Copyright 2020 Google Inc.
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

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v72/github"
	"golang.org/x/oauth2"
)

type GitHubProvider struct {
	client *github.Client
}

func (p *GitHubProvider) getListOptions(m ListOptions) github.ListOptions {
	return github.ListOptions{
		Page:    m.Page,
		PerPage: m.PerPage,
	}
}

func (p *GitHubProvider) getIssues(i []*github.Issue) []*Issue {
	r := make([]*Issue, len(i))
	for k, v := range i {
		m := Issue{}
		b, err := json.Marshal(v)
		if err != nil {
			fmt.Println(err)
		}
		err = json.Unmarshal(b, &m)
		if err != nil {
			fmt.Println(err)
		}
		r[k] = &m
	}
	return r
}

func (p *GitHubProvider) getRate(i *github.Rate) Rate {
	r := Rate{}
	b, err := json.Marshal(i)
	if err != nil {
		fmt.Println(err)
	}
	err = json.Unmarshal(b, &r)
	if err != nil {
		fmt.Println(err)
	}
	return r
}

func (p *GitHubProvider) getResponse(i *github.Response) *Response {
	if i == nil {
		return nil
	}
	r := Response{
		NextPage:      i.NextPage,
		PrevPage:      i.PrevPage,
		FirstPage:     i.FirstPage,
		LastPage:      i.LastPage,
		NextPageToken: i.NextPageToken,
		Rate:          p.getRate(&(*i).Rate),
	}
	return &r
}

func (p *GitHubProvider) getIssueListByRepoOptions(sp SearchParams) *github.IssueListByRepoOptions {
	return &github.IssueListByRepoOptions{
		ListOptions: p.getListOptions(sp.IssueListByRepoOptions.ListOptions),
		State:       sp.IssueListByRepoOptions.State,
		Since:       sp.IssueListByRepoOptions.Since,
	}
}

func (p *GitHubProvider) IssuesListByRepo(ctx context.Context, sp SearchParams) (i []*Issue, r *Response, err error) {
	opt := p.getIssueListByRepoOptions(sp)
	gi, gr, err := p.client.Issues.ListByRepo(ctx, sp.Repo.Organization, sp.Repo.Project, opt)
	i = p.getIssues(gi)
	r = p.getResponse(gr)
	return
}

func (p *GitHubProvider) getIssuesListCommentsOptions(sp SearchParams) *github.IssueListCommentsOptions {
	return &github.IssueListCommentsOptions{
		ListOptions: p.getListOptions(sp.IssueListCommentsOptions.ListOptions),
	}
}

func (p *GitHubProvider) getIssueComments(i []*github.IssueComment) []*IssueComment {
	r := make([]*IssueComment, len(i))
	for k, v := range i {
		m := IssueComment{}
		b, err := json.Marshal(v)
		if err != nil {
			fmt.Println(err)
		}
		err = json.Unmarshal(b, &m)
		if err != nil {
			fmt.Println(err)
		}
		r[k] = &m
	}
	return r
}

func (p *GitHubProvider) IssuesListComments(ctx context.Context, sp SearchParams) (i []*IssueComment, r *Response, err error) {
	opt := p.getIssuesListCommentsOptions(sp)
	gc, gr, err := p.client.Issues.ListComments(ctx, sp.Repo.Organization, sp.Repo.Project, sp.IssueNumber, opt)
	i = p.getIssueComments(gc)
	r = p.getResponse(gr)
	return
}

func (p *GitHubProvider) getIssuesListIssueTimelineOptions(sp SearchParams) *github.ListOptions {
	return &github.ListOptions{
		PerPage: sp.ListOptions.PerPage,
	}
}

func (p *GitHubProvider) getIssueTimeline(i []*github.Timeline) []*Timeline {
	r := make([]*Timeline, len(i))
	for k, v := range i {
		m := Timeline{}
		b, err := json.Marshal(v)
		if err != nil {
			fmt.Println(err)
		}
		err = json.Unmarshal(b, &m)
		if err != nil {
			fmt.Println(err)
		}
		r[k] = &m
	}
	return r
}

func (p *GitHubProvider) IssuesListIssueTimeline(ctx context.Context, sp SearchParams) (i []*Timeline, r *Response, err error) {
	opt := p.getIssuesListIssueTimelineOptions(sp)
	it, ir, err := p.client.Issues.ListIssueTimeline(ctx, sp.Repo.Organization, sp.Repo.Project, sp.IssueNumber, opt)
	i = p.getIssueTimeline(it)
	r = p.getResponse(ir)
	return
}

func (p *GitHubProvider) getPullRequestsListOptions(sp SearchParams) *github.PullRequestListOptions {
	return &github.PullRequestListOptions{
		ListOptions: p.getListOptions(sp.PullRequestListOptions.ListOptions),
		State:       sp.PullRequestListOptions.State,
		Sort:        sp.PullRequestListOptions.Sort,
		Direction:   sp.PullRequestListOptions.Direction,
	}
}

func (p *GitHubProvider) getPullRequestsList(i []*github.PullRequest) []*PullRequest {
	r := make([]*PullRequest, len(i))
	for k, v := range i {
		m := PullRequest{}
		b, err := json.Marshal(v)
		if err != nil {
			fmt.Println(err)
		}
		err = json.Unmarshal(b, &m)
		if err != nil {
			fmt.Println(err)
		}
		r[k] = &m
	}
	return r
}

func (p *GitHubProvider) PullRequestsList(ctx context.Context, sp SearchParams) (i []*PullRequest, r *Response, err error) {
	opt := p.getPullRequestsListOptions(sp)
	gpr, gr, err := p.client.PullRequests.List(ctx, sp.Repo.Organization, sp.Repo.Project, opt)
	i = p.getPullRequestsList(gpr)
	r = p.getResponse(gr)
	return
}

func (p *GitHubProvider) getPullRequest(i *github.PullRequest) *PullRequest {
	r := PullRequest{}
	b, err := json.Marshal(i)
	if err != nil {
		fmt.Println(err)
	}
	err = json.Unmarshal(b, &r)
	if err != nil {
		fmt.Println(err)
	}
	return &r
}

func (p *GitHubProvider) PullRequestsGet(ctx context.Context, sp SearchParams) (i *PullRequest, r *Response, err error) {
	pr, gr, err := p.client.PullRequests.Get(ctx, sp.Repo.Organization, sp.Repo.Project, sp.IssueNumber)
	i = p.getPullRequest(pr)
	r = p.getResponse(gr)
	return
}

func (p *GitHubProvider) getPullRequestListComments(i []*github.PullRequestComment) []*PullRequestComment {
	r := make([]*PullRequestComment, len(i))
	for k, v := range i {
		m := PullRequestComment{}
		b, err := json.Marshal(v)
		if err != nil {
			fmt.Println(err)
		}
		err = json.Unmarshal(b, &m)
		if err != nil {
			fmt.Println(err)
		}
		r[k] = &m
	}
	return r
}

func (p *GitHubProvider) getPullRequestsListCommentsOptions(sp SearchParams) *github.PullRequestListCommentsOptions {
	return &github.PullRequestListCommentsOptions{
		ListOptions: p.getListOptions(sp.ListOptions),
	}
}

func (p *GitHubProvider) PullRequestsListComments(ctx context.Context, sp SearchParams) (i []*PullRequestComment, r *Response, err error) {
	opt := p.getPullRequestsListCommentsOptions(sp)
	pr, gr, err := p.client.PullRequests.ListComments(ctx, sp.Repo.Organization, sp.Repo.Project, sp.IssueNumber, opt)
	i = p.getPullRequestListComments(pr)
	r = p.getResponse(gr)
	return
}

func (p *GitHubProvider) getPullRequestsListReviews(i []*github.PullRequestReview) []*PullRequestReview {
	r := make([]*PullRequestReview, len(i))
	for k, v := range i {
		m := PullRequestReview{}
		b, err := json.Marshal(v)
		if err != nil {
			fmt.Println(err)
		}
		err = json.Unmarshal(b, &m)
		if err != nil {
			fmt.Println(err)
		}
		r[k] = &m
	}
	return r
}

func (p *GitHubProvider) PullRequestsListReviews(ctx context.Context, sp SearchParams) (i []*PullRequestReview, r *Response, err error) {
	opt := p.getListOptions(sp.ListOptions)
	pr, gr, err := p.client.PullRequests.ListReviews(ctx, sp.Repo.Organization, sp.Repo.Project, sp.IssueNumber, &opt)
	i = p.getPullRequestsListReviews(pr)
	r = p.getResponse(gr)
	return
}

func NewGitHub(ctx context.Context, token string, url string) (Provider, error) {
	o := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))

	if url != "" {
		client, err := github.NewEnterpriseClient(url, url, o)
		if err != nil {
			return nil, fmt.Errorf("NewEnterpriseClient: %v", err)
		}
		return &GitHubProvider{client: client}, nil
	}
	return &GitHubProvider{client: github.NewClient(o)}, nil
}
