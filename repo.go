package main

import (
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

func (r Repo) clone() error {
	// do nothing if already exists
	_, err := os.Stat(r.path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	cmd := exec.Command("git", "clone", string(r.url), r.path)
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (r Repo) fetch() error {
	_, err := os.Stat(r.path)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "fetch")
	cmd.Dir = r.path
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (r Repo) checkout(branch string) (Path, error) {
	target, err := ioutil.TempDir("", branch)
	if err != nil {
		return "", fmt.Errorf("cannot create tmp dir: %v", err)
	}
	defer os.RemoveAll(target)
	err = copy.Copy(r.path, target)
	if err != nil {
		return "", fmt.Errorf("cannot copy repo: %v", err)
	}

	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = target
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("cannot checkout: %v", err)
	}
	return target, nil
}

func (r Repo) Checkout(branch string) (Path, error) {
	err := r.clone()
	if err != nil {
		return "", err
	}
	err = r.fetch()
	if err != nil {
		return "", err
	}
	return r.checkout(branch)
}
