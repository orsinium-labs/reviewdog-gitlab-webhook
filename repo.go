package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/otiai10/copy"
)

type Path = string

type URL string

func (url URL) Hash() (string, error) {
	h := md5.New()
	_, err := io.WriteString(h, string(url))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

type Repo struct {
	url  URL
	path Path
}

func NewRepo(root Path, url URL) (*Repo, error) {
	h, err := url.Hash()
	if err != nil {
		return nil, err
	}
	r := Repo{
		url:  url,
		path: path.Join(root, h),
	}
	return &r, nil
}

func (r Repo) Clone() error {
	// do nothing if already exists
	_, err := os.Stat(r.path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("cannot get stat on repo: %v", err)
	}

	cmd := exec.Command("git", "clone", string(r.url), r.path)
	cmd.Env = append(os.Environ(), "GIT_LFS_SKIP_SMUDGE=1")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("cannot do git clone: %v", err)
	}
	return nil
}

func (r Repo) Fetch() error {
	_, err := os.Stat(r.path)
	if err != nil {
		return fmt.Errorf("cannot get stat on repo: %v", err)
	}

	err = r.run(r.path, "git", "pull", "origin")
	if err != nil {
		return fmt.Errorf("cannot do git pull: %v", err)
	}
	return nil
}

func (r Repo) Checkout(sha string) (Path, error) {
	targetRepo, err := ioutil.TempDir("", sha)
	if err != nil {
		return "", fmt.Errorf("cannot create tmp dir: %v", err)
	}
	err = copy.Copy(r.path, targetRepo)
	if err != nil {
		return "", fmt.Errorf("cannot copy repo: %v", err)
	}

	err = r.run(targetRepo, "git", "checkout", "--detach", sha)
	if err != nil {
		return "", fmt.Errorf("cannot do git checkout: %v", err)
	}

	return targetRepo, nil
}

func (Repo) run(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("cannot run command: %v: %s", err, stderr.String())
	}
	return nil
}
