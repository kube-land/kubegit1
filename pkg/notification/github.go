package notification

import(
  "encoding/json"
  "net/http"
  "bytes"
  "io/ioutil"
  "fmt"
  "strings"
  "k8s.io/klog"
)

type GithubStatus struct {
  State        string `json:"state,omitempty"`
  TargetURL    string `json:"target_url,omitempty"`
  Description  string `json:"description,omitempty"`
  Context      string `json:"context,omitempty"`
}

func (g GithubConfig) Notify(status string, kind string, namespace string, name string, annotations map[string]string) error {

  var state string
  var description string

  if status == "STARTED" {
    state = "pending"
    description = "The build has started"
  } else if status == "SUCCEEDED" {
    state = "success"
    description = "The build has succeeded"
  } else if status == "FAILED" {
    state = "failure"
    description = "The build has failed"
  }

  githubStatus := GithubStatus{
      State: state,
      TargetURL: renderURL(g.URL, namespace, name),
      Description: description,
      Context: "kube-git/build",
    }

  repository := annotations["kubegit.appwavelets.com/repository"]
  owner, repo := parseGithubRepository(repository)
  if owner == "" || repo == "" {
    return fmt.Errorf("Could't parse GitHub repository or owner: %s", repository)
  }

  if g.API == "" {
    g.API = "https://api.github.com"
  }
  apiURL := fmt.Sprintf("%s/repos/%s/%s/statuses/%s", g.API, owner, repo, annotations["kubegit.appwavelets.com/commit"])
  return githubStatus.SendGithubStatus(g.Token, apiURL)
}

func (s GithubStatus) SendGithubStatus(token string, apiURL string) error {

  klog.Infof("Send GitHub status: %s", apiURL)

  requestByte, err := json.Marshal(s)
  if err != nil {
    return err
  }

  req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestByte))
  if err != nil {
    return err
  }
  req.Header.Set("Content-Type", "application/json")
  req.SetBasicAuth(token, "x-oauth-basic")

  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
    return err
  }

  defer resp.Body.Close()

  _, err = ioutil.ReadAll(resp.Body)
  if err != nil {
    return err
  }
  return nil

}

func parseGithubRepository(repository string) (string, string) {
  // https://github.com/appwavelets/kube-git.git
  // git@github.com:appwavelets/kube-git.git
  repoTokenize := strings.Split(repository, "/")
  if len(repoTokenize) >= 2 {
    owner := repoTokenize[len(repoTokenize)-2]
    if len(strings.Split(owner, ":")) == 2 {
      owner = strings.Split(owner, ":")[1]
    }
    repo := strings.TrimSuffix(repoTokenize[len(repoTokenize)-1], ".git")
    return owner, repo
  }
  return "", ""
}
