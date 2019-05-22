package git

import (
	"io/ioutil"

	"golang.org/x/crypto/ssh"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	//"gopkg.in/src-d/go-git.v4/storage/memory"
	//"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"k8s.io/klog"
	"os"
	"path/filepath"
	"fmt"
	"os/exec"

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
	fmt.Println(err)
	fmt.Println(output)

	b, err := ioutil.ReadFile(filepath.Join(path, manifest))

	return b, nil

}
