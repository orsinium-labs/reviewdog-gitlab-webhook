package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/francoispqt/onelog"
	"github.com/tidwall/gjson"
)

type Server struct {
	Address string
	Repos   Path
	Secret  string
	Token   string
	BaseURL string `toml:"base_url"`
	logger  *onelog.Logger
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
	// get repo url and branch name
	json := gjson.Parse(string(body))
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
	repo, err := NewRepo(s.Repos, URL(url))
	if err != nil {
		s.logger.ErrorWith("cannot make repo").Err("error", err).Write()
		return
	}
	path, err := repo.Checkout(branch)
	if err != nil {
		s.logger.ErrorWith("cannot checkout").Err("error", err).Write()
		return
	}

	reviewer := Reviewer{
		Path:    path,
		Token:   s.Token,
		BaseURL: s.BaseURL,
	}
	reviewer = reviewer.ForPR(json)

	log := s.logger.ErrorWith("cannot review").String("repo", reviewer.Repo)

	err = reviewer.Flake8()
	if err != nil {
		log.String("tool", "flake8").Err("error", err).Write()
		return
	}

}

func main() {
	var s Server
	_, err := toml.DecodeFile("config.toml", &s)
	if err != nil {
		log.Fatal(err)
	}
	s.logger = onelog.New(
		os.Stdout,
		onelog.ALL,
	)
	http.HandleFunc("/review", s.Handle)
	s.logger.InfoWith("listening").String("address", s.Address).Write()
	log.Fatal(http.ListenAndServe(s.Address, nil))
}
