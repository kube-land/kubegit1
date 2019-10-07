# kube-git

The kube-git controller allows defining `GitHook` object on Kubernetes to trigger resources that run to completion. The main goal is to make CI/CD more easier on kubernetes. Currently supported resources is:

* Argo Workflows
* Kubernetes Jobs

Supported SCM that will trigger the resource based on push events:
* GitHub

Supported notifications:
* GitHub build status (by personal access tokens)
* Slack

## Installation

To configure the controller change `install/deployment.yaml` and `install/service.yaml` to fit your environment.

To use `kube-git` you have to expose the controller publicly to act as a webhook for GitHub. A secret should be configured with controller deployment and used from by GitHub to secure the webhook. Configure either `github-webhook-secret` argument or `GITHUB_WEBHOOK_SECRET` environment variable.

To configure the notification, the argument `-notification-config-file` of the controller should be configred with YAML file (eg., `etc/kube-git/notification.yaml`):

```yaml
# notification.yaml
github:
  github-example:
    # notification URL
    url: "https://argo.example.com/workflows/${NAMESPACE}/${NAME}"
    # personal access token
    token: ...
  github-example-1:
    ...
slack:
  slack-ci:
    url: "https://k8s.example.com/#!/job/${NAMESPACE}/${NAME}?namespace=${NAMESPACE}"
    webhookURL: ...
    channel: "#ci"
  slack-ci-1:
    ...
```

Using variables `${NAMESPACE}` and `${NAME}` in url will be replaced by `kube-git` with the resource (Job/Workflow) namespace and name.

Finally install the controller (after configuration):

```bash
kubectl apply -k install
```

## Usage

There is a simple example in `examples` for `kube-git` CI.

To configure GitHub webhook use `DOMAIN/github` URL (for example: `https://kubegit.example.com/github`) and `application/json` content type.

Then you deploy a `GitHook`. For example:

```yaml
apiVersion: kubegit.appspero.com/v1alpha1
kind: GitHook
metadata:
  name: githook-example
  namespace: ci
spec:
  # for SSH
  repository: git@github.com:appspero/kube-git.git
  # for http
  #repository: https://github.com/appspero/kube-git.git
  branches:
    - "*"
  # manifest that will be applied
  manifest: argo.yaml
  # append timestamp to resource name
  #timestampSuffix: true
  argoWorkflow:
    revisionParameterName: revision
    branchParameterName: branch
  #usernameSecret:
  #  name: secret-example
  #  key: username
  #passwordSecret:
  #  name: secret-example
  #  key: password
  sshPrivateKeySecret:
    name: secret-example
    key: key
  notification:
    github: github-example
    slack: slack-example
```

The manifest file should have either one `Workflow` or one `Job`. If the defined manifest is of type Argo `Workflow`, you can use `argoWorkflow.revisionParameterName` and `argoWorkflow.branchParameterName` to substitute `arguments.parameters` in the Workflow . That could be used to apply conditions on branches, or to checkout the repository revision of the commit that triggered the Workflow.

When you specify branches in `GitHook` you can use wildcard names or specfic names which should be full ref name of git branch (`refs/heads/BRANCH_NAME`). Further the `argoWorkflow.branchParameterName` will be replaced by the full ref name of the git branch.

*Note:* It is recommended to use `generateName` instead of `name` for the defined resource (Job/Workflow) in the manifest file. If `generateName` is not used, you can set `timestampSuffix: true` to append timestamp to resource name.

## Build

```bash
GOPATH=~/go CODEGEN_PKG=~/go/src/k8s.io/code-generator bash -xe hack/update-codegen.sh
docker build -t abdullahalmariah/kube-git:latest .
docker push abdullahalmariah/kube-git:latest
```

## Memory Utilization

Originally we have used `memfs` form `gopkg.in/src-d/go-billy.v4/memfs` to clone the repository that have a push event to get the manifest file. If the repository size is not small (like `kube-git` which is bigger than 100MB), the controller will have a high memory utilization. To reduce the memory usage the clone behaviour has changed to plain clone (`gopkg.in/src-d/go-git.v4`) to tmp directory and then we fetch the manifest file.

To analyse the heap memory of the controller we used `pprof` by adding it to `cmd/kubegit/main.go`:

```go
import _ "net/http/pprof"
import (
  "log"
  ...
)
...
go func() {
  log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Then enable the port-forwarding:

```bash
kubectl port-forward kube-git-... 6060:6060
```

Then we profile the memory as follow:

```bash
go tool pprof -alloc_space http://localhost:6060/debug/pprof/heap
```

The previous command will output the `pb.gz` profileing data which could be viewed as follow:

```bash
go tool pprof -http=:8090 /path/to/<FILE_NAME>.pb.gz
```

## TODO
* Support leader election for high-availability mode
* Adding support for more notification
* Adding support for Bitbucket
* Controller shutdown
