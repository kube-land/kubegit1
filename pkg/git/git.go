package git

import (
	"io/ioutil"

	"golang.org/x/crypto/ssh"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4/plumbing"

)

func FetchGitFile(repository string, branch string, username []byte, password []byte, key []byte, hash string, manifest string) ([]byte, error) {

	var r *git.Repository
	var err error
	fs := memfs.New()

	if len(key) != 0 {

		// future: ParsePrivateKeyWithPassphrase
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		// InsecureIgnoreHostKey should be replaced by adding hostkey to config
		auth := &gitssh.PublicKeys{User: "git", Signer: signer}
		auth.HostKeyCallbackHelper.HostKeyCallback = ssh.InsecureIgnoreHostKey()

		r, err = git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
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
		r, err = git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
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

		r, err = git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
	    URL: repository,
			ReferenceName: plumbing.ReferenceName(branch),
			SingleBranch: true,
			NoCheckout: true,
		})
		if err != nil {
			return nil, err
		}

	}

	worktree, err := r.Worktree()
	if err != nil {
		return nil, err
	}
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(hash),
	})
	if err != nil {
		return nil, err
	}

	a, err := worktree.Filesystem.Open(manifest)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(a)

	return b, nil

}
