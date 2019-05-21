package notification

import(
  "encoding/json"
  "net/http"
  "bytes"
  "io/ioutil"
  "fmt"
  "strings"
)

type GithubStatus struct {
  State        string `json:"state,omitempty"`
  TargetURL    string `json:"target_url,omitempty"`
  Description  string `json:"description,omitempty"`
  Context      string `json:"context,omitempty"`
}

func (g GithubConfig) Notify(status string, kind string, namespace string, name string, annotations map[string]string) {

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

  owner, repo := parseGithubRepository(annotations["kubegit.appwavelets.com/repository"])

  if g.API == "" {
    g.API = "https://api.github.com"
  }
  apiURL := fmt.Sprintf("%s/repos/%s/%s/statuses/%s", g.API, owner, repo, annotations["kubegit.appwavelets.com/commit"])
  githubStatus.SendGithubStatus(g.Token, apiURL)
}

func (s GithubStatus) SendGithubStatus(token string, apiURL string) {

  fmt.Println(token, apiURL)
  fmt.Println(s)

  requestByte, err := json.Marshal(s)
  if err != nil {
    fmt.Println(err)
  }

  req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestByte))
  if err != nil {
    fmt.Println(err)
  }
  req.Header.Set("Content-Type", "application/json")
  req.SetBasicAuth(token, "x-oauth-basic")

  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
    fmt.Println(err)
  }

  defer resp.Body.Close()

  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {

  }


  if string(body) == "missing_text_or_fallback_or_attachments" && resp.StatusCode == 400 {

  } else if string(body) == "invalid_payload" && resp.StatusCode == 400 {

  } else if string(body) == "channel_not_found" && resp.StatusCode == 404 {

  } else {

  }

  fmt.Println(resp.StatusCode)
  fmt.Println(string(body))

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
