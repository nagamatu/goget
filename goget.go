package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	iso8601Format = "2006-01-02T15:04:05-07:00"
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

func dependPackageNameListByFiles(dir string) ([]string, error) {
	cmd := exec.Command("find", dir, "-name", "*.go", "-print")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, string(out)))
	}
	files := []string{}
	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		line := scanner.Text()
		files = append(files, line)
	}

	duplicateCheck := make(map[string]bool)
	pkgs := []string{}

	fset := token.NewFileSet()
	for _, f := range files {
		ast, err := parser.ParseFile(fset, f, nil, parser.ImportsOnly)
		if err != nil {
			continue
		}
		for _, i := range ast.Imports {
			pkg := i.Path.Value
			pkg = strings.TrimSuffix(strings.TrimPrefix(pkg, "\""), "\"")
			if duplicateCheck[pkg] {
				continue
			}
			pkgs = append(pkgs, pkg)
			duplicateCheck[pkg] = true
		}
	}

	return pkgs, nil
}

func dependPackageNameList(dir, tags string) ([]string, error) {
	args := []string{"list", "-f", `{{join .Deps "\n"}}`}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	o, err := cmd.CombinedOutput()
	if err != nil {
		// if go list doesn't work, try to use parser.ParseDir instead
		return dependPackageNameListByFiles(dir)
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
		if !strings.Contains(ss[0], ".") {
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

func goGet(gopath, slug string) error {
	cmd := exec.Command("go", "get", slug)
	cmd.Dir = gopath
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GO111MODULE=off")
	_, _ = cmd.CombinedOutput()
	return nil
}

func gitClone(gopath, slug string) error {
	ss := strings.Split(slug, "/")
	if ss[0] != "github.com" {
		return goGet(gopath, slug)
	}
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

var errAlreadyExists = errors.New("already exists")

func goget(gopath, slug string, t *time.Time) error {
	if _, err := os.Stat(path.Join(gopath, "src", slug)); !os.IsNotExist(err) {
		return errors.Wrap(errAlreadyExists, path.Join(gopath, "src", slug))
	}

	if err := prepareDirectory(gopath, slug); err != nil {
		return err
	}

	fmt.Printf("%s\n", slug)
	if err := gitClone(gopath, slug); err != nil {
		return err
	}

	commitID, err := commitIDForTime(gopath, slug, t)
	if err != nil {
		return err
	}

	return gitReset(gopath, slug, commitID)
}

func gogetAll(gopath, dir string, md *time.Time) error {
	list, err := dependPackageNameList(dir, "")
	if err != nil {
		return nil
	}

	slugs := dependSlugList(list)
	for _, slug := range slugs {
		if err := goget(gopath, slug, md); err != nil {
			if errors.Cause(err) != errAlreadyExists {
				fmt.Fprintf(os.Stderr, "%+v\n", err)
			}
			continue
		}
		if err := gogetAll(gopath, path.Join(gopath, "src", slug), md); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}
	return nil
}

func doGet(dir, gopath, slug string) error {
	md, err := lastModifiedDate(dir, ".")
	if err != nil {
		return err
	}

	if slug != "" {
		if err := goget(gopath, slug, md); err != nil {
			return err
		}
		dir = filepath.Join(gopath, "src", slug)
	}
	return gogetAll(gopath, dir, md)
}

func main() {
	var dir string
	var slug string
	flag.StringVar(&slug, "slug", "", "slug for repository (optional)")
	flag.StringVar(&dir, "dir", ".", "directory for dependency")
	flag.Parse()

	gopath, ok := os.LookupEnv("GOPATH")
	if !ok {
		fmt.Fprintf(os.Stderr, "error: GOPATH not defined")
		os.Exit(1)
	}

	if dir == "." && slug == "" {
		if len(os.Args) > 1 {
			dir = os.Args[1]
		}
	}

	if err := doGet(dir, gopath, slug); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
}
