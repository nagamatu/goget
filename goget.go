package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	sh                              = "/bin/sh"
	iso8601Format                   = "2006-01-02T15:04:05-07:00"
	getDependPackageNameListCommand = `go list -f '{{join .Deps "\n"}}' "$@"`
)

func lastModifiedDate(dir, fileName string) (*time.Time, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%aI", "--", fileName)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, string(out)))
	}
	t, err := time.Parse(iso8601Format, strings.TrimSpace(string(out)))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &t, nil
}

func dependPackageNameList(dir, tags string) ([]string, error) {
	var args []string
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	shArgs := append([]string{"-c", getDependPackageNameListCommand, sh}, args...)
	cmd := exec.Command(sh, shArgs...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Env, os.ExpandEnv("GOPATH=${GOPATH}"))
	cmd.Env = append(cmd.Env, os.ExpandEnv("PATH=${PATH}"))
	cmd.Env = append(cmd.Env, os.ExpandEnv("HOME=${HOME}"))
	o, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, string(o)))
	}
	if len(o) == 0 {
		return nil, nil
	}
	return strings.Split(strings.TrimSpace(string(o)), "\n"), nil
}

func dependSlugList(pkgs []string) []string {
	slugMap := make(map[string]bool)
	for _, pkg := range pkgs {
		ss := strings.Split(pkg, "/")
		if len(ss) < 3 {
			continue
		}
		// only support repository for github.com.
		// this skips standard package with three hierarchy layer.
		if ss[0] != "github.com" {
			continue
		}
		slugMap[path.Join(ss[0], ss[1], ss[2])] = true
	}
	var slugs []string = nil
	for slug := range slugMap {
		slugs = append(slugs, slug)
	}
	return slugs
}

func commitIDForTime(gopath, slug string, t *time.Time) (string, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%H", "--before", t.Format(iso8601Format))
	cmd.Dir = path.Join(gopath, "src", slug)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.WithStack(errors.Wrap(err, string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func gitReset(gopath, slug, commitID string) error {
	cmd := exec.Command("git", "reset", "--hard", commitID)
	cmd.Dir = path.Join(gopath, "src", slug)
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.WithStack(errors.Wrap(err, string(out)))
	}
	return nil
}

func gitClone(gopath, slug string) error {
	dir, _ := path.Split(slug)
	cmd := exec.Command("git", "clone", fmt.Sprintf("https://%s.git", slug))
	cmd.Dir = path.Join(gopath, "src", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.WithStack(errors.Wrap(err, string(out)))
	}

	return nil
}

func prepareDirectory(gopath, slug string) error {
	dir, _ := path.Split(slug)
	if _, err := os.Stat(path.Join(gopath, "src", dir)); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Join(gopath, "src", dir), 0700); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func goget(gopath, slug string, t *time.Time) error {
	if _, err := os.Stat(path.Join(gopath, "src", slug)); !os.IsNotExist(err) {
		fmt.Printf("already exist: %s\n", path.Join(gopath, "src", slug))
		return nil
	}

	if err := prepareDirectory(gopath, slug); err != nil {
		return err
	}

	if err := gitClone(gopath, slug); err != nil {
		return err
	}

	commitID, err := commitIDForTime(gopath, slug, t)
	if err != nil {
		return err
	}

	return gitReset(gopath, slug, commitID)
}

func gogetAll(gopath, dir string) error {
	md, err := lastModifiedDate(dir, ".")
	if err != nil {
		return err
	}

	list, err := dependPackageNameList(dir, "")
	if err != nil {
		return err
	}

	slugs := dependSlugList(list)
	for _, slug := range slugs {
		fmt.Printf("%s\n", slug)
		if err := goget(gopath, slug, md); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
	}
	return nil
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	gopath, ok := os.LookupEnv("GOPATH")
	if !ok {
		fmt.Fprintf(os.Stderr, "error: GOPATH not defined")
		os.Exit(1)
	}

	if err := gogetAll(gopath, dir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
