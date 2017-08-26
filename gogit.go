package gogit

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// GitRepo is the basic store object.
type GitRepo struct {
	repo   string
	folder string
	logger *logrus.Logger
	log    *logrus.Entry
}

// New returns a new GPGStore that can then needs to be initialized with Init()
// The `repo` is the supplied git repository. Optionally you can include
// a string for which folder to save the repo to. The default location is the
// current directory.
func New(repo string, optionalFolder ...string) (*GitRepo, error) {
	var err error
	gr := new(GitRepo)
	gr.repo = repo
	if len(optionalFolder) > 0 {
		gr.folder = optionalFolder[0]
	} else {
		gr.folder = ParseRepoFolder(repo)
	}
	gr.folder, err = filepath.Abs(gr.folder)
	if err != nil {
		return gr, err
	}
	if !exists(gr.folder) {
		err = os.MkdirAll(gr.folder, 0775)
		if err != nil {
			return gr, err
		}
	}
	gr.logger = logrus.New()
	gr.log = gr.logger.WithFields(logrus.Fields{
		"source": "gogit",
	})
	gr.logger.SetLevel(logrus.WarnLevel)
	return gr, nil
}

// Debug will allow verbose output if enabled.
func (gr *GitRepo) Debug(on bool) {
	if on {
		gr.logger.SetLevel(logrus.InfoLevel)
	} else {
		gr.logger.SetLevel(logrus.WarnLevel)
	}
}

// Update will clone a repo if it doesn't exist or pull a repo, if it does.
func (gr *GitRepo) Update() (err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	defer os.Chdir(cwd)
	err = os.Chdir(gr.folder)
	if err != nil {
		return
	}
	var cmd *exec.Cmd
	var stdoutStderr []byte
	pullOrClone := ""
	if !exists(path.Join(gr.folder, ".git")) {
		gr.log.Infof("Running: git clone %s %s", gr.repo, ".")
		cmd = exec.Command("git", "clone", gr.repo, ".")
		pullOrClone = "clone"
	} else {
		gr.log.Info("Running: git pull --rebase origin master")
		cmd = exec.Command("git", "pull", "--rebase", "origin", "master")
		pullOrClone = "pull"
	}
	stdoutStderr, err = cmd.CombinedOutput()
	gr.log.Infof("Output: [%s]\n", stdoutStderr)
	if bytes.Contains(stdoutStderr, []byte("fatal")) {
		err = errors.New("Could not " + pullOrClone + " repo")
	}
	return
}

// Push will push the repo to the master branch.
func (gr *GitRepo) Push() (err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	defer os.Chdir(cwd)
	err = os.Chdir(gr.folder)
	if err != nil {
		return
	}

	cmd := exec.Command("git", "push", "origin", "master")
	gr.log.Info("git push origin master")
	stdoutStderr, err := cmd.CombinedOutput()
	gr.log.Infof("Output: [%s]\n", stdoutStderr)
	if bytes.Contains(stdoutStderr, []byte("error")) {
		err = errors.New(string(stdoutStderr))
		return
	}
	return
}

// AddData will write the `data` to a new file, `fp` in the repo
// and then perform a commit with the message of that files name.
// Note that the filename should be respective to the root of the
// repository.
func (gr *GitRepo) AddData(data []byte, fp string) (err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	defer os.Chdir(cwd)
	err = os.Chdir(gr.folder)
	if err != nil {
		return
	}
	dir, file := filepath.Split(fp)
	gr.log.Infof("Got file '%s' in path '%s'", file, dir)
	if len(dir) > 0 {
		gr.log.Infof("Created directory %s", dir)
		err = os.MkdirAll(dir, 0775)
		if err != nil {
			return
		}
	}
	err = ioutil.WriteFile(fp, data, 0755)
	if err != nil {
		return err
	}
	gr.log.Infof("Wrote %d bytes", len(data))

	cmd := exec.Command("git", "add", fp)
	gr.log.Info("git", "add", fp)
	stdoutStderr, err := cmd.CombinedOutput()
	gr.log.Infof("Output: [%s]\n", stdoutStderr)
	if bytes.Contains(stdoutStderr, []byte("error")) {
		err = errors.New(string(stdoutStderr))
		return
	}

	_, fileName := filepath.Split(fp)
	cmd = exec.Command("git", "commit", "-m", "Add "+fileName, fp)
	gr.log.Info("git", "commit", "-am", "Add "+fileName, fp)
	stdoutStderr, _ = cmd.CombinedOutput()
	gr.log.Infof("Output: [%s]\n", stdoutStderr)
	if bytes.Contains(stdoutStderr, []byte("error")) {
		err = errors.New(string(stdoutStderr))
		return
	}
	return
}

func GetRemoteOriginURL(repoFolder string) (repo string, err error) {
	// move to repo
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	defer os.Chdir(cwd)
	err = os.Chdir(repoFolder)
	if err != nil {
		return
	}

	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	stdoutStderr, err := cmd.CombinedOutput()
	repo = strings.TrimSpace(string(stdoutStderr))
	return
}

func ParseRepoFolder(repo string) (folder string) {
	firstPart := strings.Split(repo, ".git")[0]
	firstPartSplit := strings.Split(firstPart, "/")
	folder = strings.TrimSpace(firstPartSplit[len(firstPartSplit)-1])
	return
}

// exists returns whether the given file or directory exists or not
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}
