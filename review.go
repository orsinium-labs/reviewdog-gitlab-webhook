package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/parser"
	gitlabservice "github.com/reviewdog/reviewdog/service/gitlab"
	"github.com/tidwall/gjson"
	"github.com/xanzy/go-gitlab"
)

type Reviewer struct {
	Path    string
	Token   string
	BaseURL string

	Owner       string
	Repo        string
	PullRequest int
	SHA         string
}

func (r Reviewer) ForPR(json gjson.Result) Reviewer {
	r.PullRequest = int(json.Get("object_attributes.iid").Int())
	r.SHA = json.Get("object_attributes.last_commit.id").String()

	repoPath := json.Get("object_attributes.target.path_with_namespace").String()
	parts := strings.SplitN(repoPath, "/", 2)
	if len(parts) != 2 {
		return r
	}
	r.Owner = parts[0]
	r.Repo = parts[1]
	return r
}

func (r Reviewer) Validate() error {
	if r.Path == "" {
		return errors.New("path is required")
	}
	if r.Token == "" {
		return errors.New("token is required")
	}
	if r.BaseURL == "" {
		return errors.New("base URL is required")
	}
	if r.Owner == "" {
		return errors.New("owner is required")
	}
	if r.Repo == "" {
		return errors.New("repo is required")
	}
	if r.SHA == "" {
		return errors.New("SHA is required")
	}
	if r.PullRequest == 0 {
		return errors.New("PR ID is required")
	}
	return nil
}

func (r Reviewer) review(format string, reader io.Reader) error {
	parser, err := parser.New(&parser.Option{FormatName: format})
	if err != nil {
		return err
	}

	client, err := gitlab.NewClient(
		r.Token,
		gitlab.WithBaseURL(r.BaseURL),
	)
	if err != nil {
		return err
	}

	comment, err := gitlabservice.NewGitLabMergeRequestDiscussionCommenter(
		client, r.Owner, r.Repo, r.PullRequest, r.SHA)
	if err != nil {
		return err
	}

	diff, err := gitlabservice.NewGitLabMergeRequestDiff(
		client, r.Owner, r.Repo, r.PullRequest, r.SHA)
	if err != nil {
		return err
	}

	dog := reviewdog.NewReviewdog(
		"Mr. Meeseeks",
		parser,
		comment,
		diff,
		filter.ModeAdded,
		false,
	)
	ctx := context.Background()
	return dog.Run(ctx, reader)
}

func (r Reviewer) Review(format string, command []string) error {
	reader, writer := io.Pipe()
	defer reader.Close()

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = r.Path
	cmd.Stdout = writer
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	var cmderr error
	go func() {
		cmderr = cmd.Run()
		err := writer.Close()
		if cmderr == nil {
			cmderr = err
		}
	}()

	err := r.review(format, reader)
	if cmderr != nil {
		return fmt.Errorf("error running command: %v: %s", cmderr, stderr.String())
	}
	if err != nil {
		return fmt.Errorf("error running reviewdog: %v", err)
	}
	return nil
}
