package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type gogetTestSuite struct {
	suite.Suite
}

func Test_gogetTestSuite(t *testing.T) {
	suite.Run(t, new(gogetTestSuite))
}

func (t *gogetTestSuite) Test_dependSlugList() {
	for _, testdata := range []struct {
		pkgs     []string
		expected []string
	}{
		{
			[]string{"github.com/stretchr/testify/suite"},
			[]string{"github.com/stretchr/testify"},
		},
		{
			[]string{"github.com/pkg"},
			nil,
		},
	} {
		actual := dependSlugList(testdata.pkgs)
		t.Assert().Equal(testdata.expected, actual)
	}
}

func (t *gogetTestSuite) Test_lastModifiedDate_falure() {
	_, err := lastModifiedDate(".", "notexists")
	t.Assert().Error(err)
}

func (t *gogetTestSuite) Test_dependPackageNameList() {
	expected := []string{"bufio", "bytes", "context", "errors", "flag", "fmt", "github.com/pkg/errors", "go/ast", "go/parser", "go/scanner", "go/token", "internal/bytealg", "internal/cpu", "internal/fmtsort", "internal/oserror", "internal/poll", "internal/race", "internal/reflectlite", "internal/syscall/execenv", "internal/syscall/unix", "internal/testlog", "internal/unsafeheader", "io", "io/ioutil", "math", "math/bits", "os", "os/exec", "path", "path/filepath", "reflect", "runtime", "runtime/internal/atomic", "runtime/internal/math", "runtime/internal/sys", "sort", "strconv", "strings", "sync", "sync/atomic", "syscall", "time", "unicode", "unicode/utf8", "unsafe"}
	actual, err := dependPackageNameList(".", "")
	t.Assert().NoError(err)
	t.Assert().Equal(expected, actual)
}

func (t *gogetTestSuite) Test_commitIDForTime() {
	gopath, err := ioutil.TempDir("", "test*")
	t.Assert().NoError(err)
	defer os.RemoveAll(gopath)

	slug := "github.com/nagamatu/goget"

	err = prepareDirectory(gopath, slug)
	t.Assert().NoError(err)
	err = gitClone(gopath, slug)
	t.Assert().NoError(err)

	tm, err := time.Parse(iso8601Format, "2021-05-07T16:17:20+09:00")
	t.Assert().NoError(err)
	actual, err := commitIDForTime(gopath, slug, &tm)
	t.Assert().NoError(err)
	t.Assert().Equal("1dc3271aaafc89687f5971bd95c2130ecac307b8", actual)
}

func (t *gogetTestSuite) Test_gitClone() {
	gopath, err := ioutil.TempDir("", "test*")
	t.Assert().NoError(err)
	defer os.RemoveAll(gopath)

	slug := "github.com/nagamatu/goget"
	err = prepareDirectory(gopath, slug)
	t.Assert().NoError(err)
	err = gitClone(gopath, slug)
	t.Assert().NoError(err)

	expected, err := time.Parse(iso8601Format, "2021-05-07T18:57:39+09:00")
	t.Assert().NoError(err)
	md, err := lastModifiedDate(path.Join(gopath, "src", slug), "README.md")
	t.Assert().NoError(err)
	t.Assert().Equal(expected, *md)
}

func (t *gogetTestSuite) Test_goget() {
	gopath, err := ioutil.TempDir("", "test*")
	t.Assert().NoError(err)
	defer os.RemoveAll(gopath)

	slug := "github.com/nagamatu/goget"
	md, err := time.Parse(iso8601Format, "2021-05-07T16:18:10+09:00")
	t.Assert().NoError(err)
	err = goget(gopath, slug, &md)
	t.Assert().NoError(err)

	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = path.Join(gopath, "src", slug)
	out, err := cmd.CombinedOutput()
	t.Assert().NoError(err)
	t.Assert().Equal("ce8cefc create README.md\n", string(out))
}

func (t *gogetTestSuite) Test_gogetAll() {
	gopath, err := ioutil.TempDir("", "test*")
	t.Assert().NoError(err)
	defer os.RemoveAll(gopath)

	dir := "."
	md, err := lastModifiedDate(dir, ".")
	t.Assert().NoError(err)
	err = gogetAll(gopath, dir, md)
	t.Assert().NoError(err)

	_, err = os.Stat(path.Join(gopath, "src", "github.com", "pkg", "errors", "stack.go"))
	t.Assert().NoError(err)

	infos, err := ioutil.ReadDir(path.Join(gopath, "src", "github.com"))
	t.Assert().NoError(err)
	t.Assert().Equal(1, len(infos))
	t.Assert().Equal("pkg", infos[0].Name())
}

func (t *gogetTestSuite) Test_dependPackageNameListByFiles() {
	pkgs, err := dependPackageNameListByFiles(".")
	t.Assert().NoError(err)
	expected := []string{
		"bufio", "bytes", "flag", "fmt", "go/parser", "go/token", "os", "os/exec", "path", "path/filepath", "strings", "time", "github.com/pkg/errors", "io/ioutil", "testing", "github.com/stretchr/testify/suite",
	}
	t.Assert().Equal(expected, pkgs)
}
