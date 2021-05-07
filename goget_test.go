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

func (t *gogetTestSuite) Test_lastModifiedDate() {
	expected, err := time.Parse(iso8601Format, "2021-05-07T16:18:10+09:00")
	t.Assert().NoError(err)
	actual, err := lastModifiedDate(".", "README.md")
	t.Assert().NoError(err)
	t.Assert().Equal(expected, *actual)
}

func (t *gogetTestSuite) Test_lastModifiedDate_falure() {
	_, err := lastModifiedDate(".", "notexists")
	t.Assert().Error(err)
}

func (t *gogetTestSuite) Test_dependPackageNameList() {
	expected := []string{"github.com/pkg/errors"}
	actual, err := dependPackageNameList("")
	t.Assert().NoError(err)
	t.Assert().Equal(expected, actual)
}

func (t *gogetTestSuite) Test_commitIDForTime() {
	gopath := os.Getenv("GOPATH")
	tm, err := time.Parse(iso8601Format, "2021-05-07T16:17:20+09:00")
	t.Assert().NoError(err)
	actual, err := commitIDForTime(gopath, "github.com/nagamatu/goget", &tm)
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

	expected, err := time.Parse(iso8601Format, "2021-05-07T16:18:10+09:00")
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

	err = gogetAll(gopath)
	t.Assert().NoError(err)

	_, err = os.Stat(path.Join(gopath, "src", "github.com", "pkg", "errors", "stack.go"))
	t.Assert().NoError(err)

	infos, err := ioutil.ReadDir(path.Join(gopath, "src", "github.com"))
	t.Assert().NoError(err)
	t.Assert().Equal(1, len(infos))
	t.Assert().Equal("pkg", infos[0].Name())
}
