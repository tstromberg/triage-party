// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hubbub

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v31/github"
	"github.com/google/triage-party/pkg/logu"
	"github.com/google/triage-party/pkg/persist"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

// cachedIssues returns issues, cached if possible
func (h *Engine) cachedIssues(ctx context.Context, org string, project string, state string, updateAge time.Duration, newerThan time.Time) ([]*github.Issue, time.Time, error) {
	key := issueSearchKey(org, project, state, updateAge)

	if x := h.cache.GetNewerThan(key, newerThan); x != nil {
		// Normally the similarity tables are only updated when fresh data is encountered.
		if newerThan.IsZero() {
			go h.updateSimilarIssues(key, x.Issues)
		}

		return x.Issues, x.Created, nil
	}

	klog.V(1).Infof("cache miss for %s newer than %s", key, logu.STime(newerThan))
	issues, created, err := h.updateIssues(ctx, org, project, state, updateAge, key)
	if err != nil {
		klog.Warningf("Retrieving stale results for %s due to error: %v", key, err)
		x := h.cache.GetNewerThan(key, time.Time{})
		if x != nil {
			return x.Issues, x.Created, nil
		}
	}
	return issues, created, err
}

// updateIssues updates the issues in cache
func (h *Engine) updateIssues(ctx context.Context, org string, project string, state string, updateAge time.Duration, key string) ([]*github.Issue, time.Time, error) {
	start := time.Now()

	opt := &github.IssueListByRepoOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		State:       state,
	}

	if updateAge != 0 {
		opt.Since = time.Now().Add(-1 * updateAge)
	}

	var allIssues []*github.Issue

	for {
		if updateAge == 0 {
			klog.Infof("Downloading %s issues for %s/%s (page %d)...", state, org, project, opt.Page)
		} else {
			klog.Infof("Downloading %s issues for %s/%s updated within %s (page %d)...", state, org, project, updateAge, opt.Page)
		}

		is, resp, err := h.client.Issues.ListByRepo(ctx, org, project, opt)

		if _, ok := err.(*github.RateLimitError); ok {
			klog.Errorf("oh snap! I reached the GitHub search API limit: %v", err)
		}

		if err != nil {
			return is, start, err
		}

		h.logRate(resp.Rate)

		for _, i := range is {
			if i.IsPullRequest() {
				continue
			}

			h.updateMtime(i, i.GetUpdatedAt())
			allIssues = append(allIssues, i)
		}

		go h.updateSimilarIssues(key, is)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	if err := h.cache.Set(key, &persist.Thing{Issues: allIssues}); err != nil {
		klog.Errorf("set %q failed: %v", key, err)
	}

	klog.V(1).Infof("updateIssues %s returning %d issues", key, len(allIssues))
	return allIssues, start, nil
}

func (h *Engine) cachedIssueComments(ctx context.Context, org string, project string, num int, newerThan time.Time, fetch bool) ([]*github.IssueComment, time.Time, error) {
	key := fmt.Sprintf("%s-%s-%d-issue-comments", org, project, num)

	if x := h.cache.GetNewerThan(key, newerThan); x != nil {
		return x.IssueComments, x.Created, nil
	}

	if !fetch {
		return nil, time.Time{}, nil
	}

	klog.V(1).Infof("cache miss for %s newer than %s", key, logu.STime(newerThan))

	comments, created, err := h.updateIssueComments(ctx, org, project, num, key)
	if err != nil {
		klog.Warningf("Retrieving stale results for %s due to error: %v", key, err)
		x := h.cache.GetNewerThan(key, time.Time{})
		if x != nil {
			return x.IssueComments, x.Created, nil
		}
	}

	return comments, created, err
}

func (h *Engine) updateIssueComments(ctx context.Context, org string, project string, num int, key string) ([]*github.IssueComment, time.Time, error) {
	klog.V(1).Infof("Downloading issue comments for %s/%s #%d", org, project, num)
	start := time.Now()

	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allComments []*github.IssueComment
	for {
		klog.Infof("Downloading comments for %s/%s #%d (page %d)...", org, project, num, opt.Page)
		cs, resp, err := h.client.Issues.ListComments(ctx, org, project, num, opt)

		if err != nil {
			return cs, start, err
		}
		h.logRate(resp.Rate)

		allComments = append(allComments, cs...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	if err := h.cache.Set(key, &persist.Thing{IssueComments: allComments}); err != nil {
		klog.Errorf("set %q failed: %v", key, err)
	}

	return allComments, start, nil
}

func toYAML(v interface{}) string {
	s, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Sprintf("yaml err: %v", err)
	}
	return strings.Replace(strings.TrimSpace(string(s)), "\n", "; ", -1)
}

func openByDefault(fs []Filter) []Filter {
	found := false
	for _, f := range fs {
		if f.State != "" {
			found = true
		}
	}
	if !found {
		fs = append(fs, Filter{State: "open"})
	}
	return fs
}

func (h *Engine) createIssueSummary(i *github.Issue, cs []*github.IssueComment, age time.Time) *Conversation {
	cl := []*Comment{}
	for _, c := range cs {
		cl = append(cl, NewComment(c))
	}

	co := h.createConversation(i, cl, age)
	r := i.GetReactions()
	co.ReactionsTotal += r.GetTotalCount()
	for k, v := range reactions(r) {
		co.Reactions[k] += v
	}
	co.ClosedBy = i.GetClosedBy()

	sort.Slice(co.Tags, func(i, j int) bool { return co.Tags[i].ID < co.Tags[j].ID })
	return co
}

// IssueSummary returns a cached conversation for an issue
func (h *Engine) IssueSummary(i *github.Issue, cs []*github.IssueComment, age time.Time) *Conversation {
	key := i.GetHTMLURL()
	cached, ok := h.seen[key]
	if ok {
		if !cached.Updated.Before(i.GetUpdatedAt()) && cached.CommentsTotal >= len(cs) {
			return h.seen[key]
		}
		klog.Infof("%s in issue cache, but was invalid. Live @ %s (%d comments), cached @ %s (%d comments)  ", i.GetHTMLURL(), i.GetUpdatedAt(), len(cs), cached.Updated, cached.CommentsTotal)
	}

	h.seen[key] = h.createIssueSummary(i, cs, age)
	return h.seen[key]
}

func isBot(u *github.User) bool {
	if u.GetType() == "bot" {
		return true
	}

	if strings.Contains(u.GetBio(), "stale issues") {
		return true
	}

	if strings.HasSuffix(u.GetLogin(), "-bot") || strings.HasSuffix(u.GetLogin(), "-robot") || strings.HasSuffix(u.GetLogin(), "_bot") || strings.HasSuffix(u.GetLogin(), "_robot") {
		return true
	}

	return false
}
