package main

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/francoispqt/onelog"
	"github.com/rakyll/statik/fs"
	"github.com/tidwall/gjson"
	_ "nico-lab.com/x/review/statik"
)

type Tool struct {
	Name    string
	Format  string
	Command []string
}

type Server struct {
	Address string
	Repos   Path
	Secret  string
	Token   string
	BaseURL string `toml:"base_url"`
	Tools   []Tool `toml:"tool"`

	logger *onelog.Logger
}

func (s Server) Handle(writer http.ResponseWriter, request *http.Request) {
	secret := request.URL.Query().Get("secret")
	if secret != s.Secret {
		s.logger.Warn("invalid secret")
		http.Error(writer, "invalid secret", http.StatusInternalServerError)
		return
	}

	if request.Method != http.MethodPost {
		s.logger.ErrorWith("unsupported method").String("method", request.Method).Write()
		http.Error(writer, "unsupported method", http.StatusInternalServerError)
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		s.logger.ErrorWith("cannot read body").Err("error", err).Write()
		http.Error(writer, "cannot read body", http.StatusInternalServerError)
		return
	}
	if len(body) == 0 {
		s.logger.Error("empty body")
		http.Error(writer, "empty body", http.StatusInternalServerError)
		return
	}

	go s.review(body)
}

func (s Server) review(body []byte) {
	json := gjson.Parse(string(body))
	kind := json.Get("object_kind").String()
	if kind != "merge_request" {
		s.logger.DebugWith("skipping").String("object_kind", kind).Write()
		return
	}
	url := json.Get("project.git_http_url").String()
	if url == "" {
		s.logger.Error("url is empty")
		return
	}
	branch := json.Get("object_attributes.source_branch").String()
	if branch == "" {
		s.logger.Error("branch is empty")
		return
	}

	// clone repo, fetch, checkout to the branch
	s.logger.InfoWith("starting repo operations").String("url", url).Write()
	repo, err := NewRepo(s.Repos, URL(url))
	if err != nil {
		s.logger.ErrorWith("cannot init Repo").Err("error", err).Write()
		return
	}
	s.logger.InfoWith("clonning").String("url", url).Write()
	err = repo.Clone()
	if err != nil {
		s.logger.ErrorWith("cannot clone").Err("error", err).Write()
		return
	}
	s.logger.InfoWith("fetching").String("url", url).Write()
	err = repo.Fetch()
	if err != nil {
		s.logger.ErrorWith("cannot fetch").Err("error", err).Write()
		return
	}
	s.logger.InfoWith("checking out").String("url", url).Write()
	path, err := repo.Checkout(branch)
	if err != nil {
		s.logger.ErrorWith("cannot checkout").Err("error", err).Write()
		return
	}
	s.logger.InfoWith("repo operations finished").String("url", url).Write()
	defer os.RemoveAll(path)

	reviewer := Reviewer{
		Path:    path,
		Token:   s.Token,
		BaseURL: s.BaseURL,
	}
	reviewer = reviewer.ForPR(json)

	for _, tool := range s.Tools {
		s.logger.InfoWith("review").
			String("repo", reviewer.Repo).
			String("tool", tool.Name).
			String("state", "running").
			Write()
		err = reviewer.Review(tool.Format, tool.Command)
		if err != nil {
			s.logger.ErrorWith("review").
				String("repo", reviewer.Repo).
				String("tool", tool.Name).
				Err("error", err).
				Write()
			return
		}
		s.logger.InfoWith("review").
			String("repo", reviewer.Repo).
			String("tool", tool.Name).
			String("state", "finished").
			Write()
	}

}

func NewServer() (*Server, error) {
	var s Server

	// read file
	statikFS, err := fs.New()
	if err != nil {
		return nil, err
	}
	r, err := statikFS.Open("/config.toml")
	if err != nil {
		return nil, err
	}
	defer r.Close()
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	_, err = toml.Decode(string(bytes), &s)
	if err != nil {
		return nil, err
	}
	s.logger = onelog.New(
		os.Stdout,
		onelog.ALL,
	)
	return &s, nil
}
