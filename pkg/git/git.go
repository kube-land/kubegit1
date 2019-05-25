package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"io/ioutil"
	"golang.org/x/crypto/ssh"

	git "gopkg.in/src-d/go-git.v4"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"k8s.io/klog"
)

func FetchGitFile(repository string, branch string, username []byte, password []byte, key []byte, hash string, manifest string) ([]byte, error) {

	var err error

	path, err := ioutil.TempDir("", hash)
	if err != nil {
		klog.Info(err)
	}

	defer os.RemoveAll(path) // clean up

	if len(key) != 0 {

		// future: ParsePrivateKeyWithPassphrase
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		// InsecureIgnoreHostKey should be replaced by adding hostkey to config
		auth := &gitssh.PublicKeys{User: "git", Signer: signer}
		auth.HostKeyCallbackHelper.HostKeyCallback = ssh.InsecureIgnoreHostKey()

		_, err = git.PlainClone(path , false, &git.CloneOptions{
	    URL: repository,
			Auth: auth,
			ReferenceName: plumbing.ReferenceName(branch),
			SingleBranch: true,
			NoCheckout: true,
		})
		if err != nil {
			return nil, err
		}

	} else if (len(username) != 0 || len(password) != 0) {

		auth := &http.BasicAuth{Username: string(username), Password: string(password)}
		_, err = git.PlainClone(path , false, &git.CloneOptions{
	    URL: repository,
			Auth: auth,
			ReferenceName: plumbing.ReferenceName(branch),
			SingleBranch: true,
			NoCheckout: true,
		})
		if err != nil {
			return nil, err
		}

	} else {

		_, err = git.PlainClone(path , false, &git.CloneOptions{
	    URL: repository,
			ReferenceName: plumbing.ReferenceName(branch),
			SingleBranch: true,
			NoCheckout: true,
		})
		if err != nil {
			return nil, err
		}

	}

	klog.Infof("Checking out revision %s", hash)
	cmd := exec.Command("git", "checkout", hash)
	cmd.Dir = path
	output, err := cmd.Output()
	klog.Info(output)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(filepath.Join(path, manifest))
	if err != nil {
		return nil, err
	}
	return b, nil

}
