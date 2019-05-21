# kube-git

The kube-git controller allows defining `GitHook` object to trigger resources that run to completion. The main goal is to make CI/CD more easier on kubernetes. Currently supported resources is:

* Argo Workflows
* Kubernetes Jobs

Supported SCM that will trigger the resource based on push events:
* GitHub

Supported notifications:
* GitHub build status (by personal access tokens)
* Slack

# Usage

There is useful example in `examples` for `kube-git` CI.

To use `kube-git` you have to expose the controller publicly to act as a webhook for GitHub. A secret should be configured with controller deployment and used from by GitHub to secure the webhook. Configure either `github-webhook-secret` argument or `GITHUB_WEBHOOK_SECRET` environment variable.

To configure GitHub webhook use `DOMAIN/github` URL (for example: `https://kubegit.example.com/github`) and `application/json` content type.

Then you deploy a `GitHook`. For example:

```yaml
apiVersion: kubegit.appwavelets.com/v1alpha1
kind: GitHook
metadata:
  name: githook-example
  namespace: ci
spec:
  # for SSH
  repository: git@github.com:appwavelets/kube-git.git
  # for http
  #repository: https://github.com/appwavelets/kube-git.git
  branches:
    - "*"
  # manifest that will be applied
  manifest: argo.yaml
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

*Note:* Make sure to use `generateName` instead of `name` for the defined resource (Job/Workflow) in the manifest file.

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

# Build

```bash
GOPATH=~/go CODEGEN_PKG=~/go/src/k8s.io/code-generator bash -xe hack/update-codegen.sh
docker build -t abdullahalmariah/kube-git:latest .
docker push abdullahalmariah/kube-git:latest
```
