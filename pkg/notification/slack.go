package notification

import(
  "encoding/json"
  "net/http"
  "bytes"
  "io/ioutil"
  "fmt"
  "k8s.io/klog"
)

type SlackMessage struct {
  Channel  string `json:"channel,omitempty"`
  Username string `json:"username,omitempty"`
  IconURL  string `json:"icon_url,omitempty"`

  Attachments []SlackAttachment `json:"attachments"`
}

type SlackAttachment struct {
  Color     string  `json:"color,omitempty"`
  Title     string  `json:"title,omitempty"`
  TitleLink string  `json:"title_link,omitempty"`
  Text      string  `json:"text,omitempty"`
  Fallback  string  `json:"fallback,omitempty"`

  Fields []SlackAttachmentField `json:"fields,omitempty"`
}

type SlackAttachmentField struct {
  Title string `json:"title,omitempty"`
  Value string `json:"value,omitempty"`
  Short bool   `json:"short,omitempty"`
}

func (s SlackConfig) Notify(status string, kind string, namespace string, name string, annotations map[string]string) error {

  var color string
  var title string
  var text string
  var fallback string

  resource := fmt.Sprintf("%s/%s", namespace, name)
  gh := annotations["kubegit.appwavelets.com/githook"]

  if status == "STARTED" {
    color = "warning"
    title = fmt.Sprintf("%s Started :large_orange_diamond:", kind)
    text = fmt.Sprintf("%s `%s` triggered by githook `%s` has started ", kind, resource, gh)
    fallback = fmt.Sprintf("kube-git: %s `%s` started", kind, resource)
  } else if status == "SUCCEEDED" {
    color = "good"
    title = fmt.Sprintf("%s Succeeded :white_check_mark:", kind)
    text = fmt.Sprintf("%s `%s` triggered by githook `%s` has succeeded ", kind, resource, gh)
    fallback = fmt.Sprintf("kube-git: %s `%s` succeeded", kind, resource)
  } else if status == "FAILED" {
    color = "danger"
    title = fmt.Sprintf("%s Failed :x:", kind)
    text = fmt.Sprintf("%s `%s` triggered by githook `%s` has failed ", kind, resource, gh)
    fallback = fmt.Sprintf("kube-git: %s `%s` failed", kind, resource)
  }

  msg := NewSlackMessage(color,
                         s.Channel,
                         renderURL(s.URL, namespace, name),
                         title,
                         text,
                         fallback,
                         annotations["kubegit.appwavelets.com/repository"],
                         annotations["kubegit.appwavelets.com/author"],
                         annotations["kubegit.appwavelets.com/branch"],
                         annotations["kubegit.appwavelets.com/commit"],
                       )
  return msg.SendSlackMessage(s.WebhookURL)
}

func (m SlackMessage) SendSlackMessage(webhookURL string) error {

  requestByte, err := json.Marshal(m)
  if err != nil {
    return err
  }

  req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(requestByte))
  if err != nil {
    return err
  }
  req.Header.Set("Content-Type", "application/json")

  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
    return err
  }

  defer resp.Body.Close()

  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return err
  }

  klog.Infof("reposnse of slack notification: %s, %s", resp.StatusCode, string(body))
  return nil

}

func NewSlackMessage(color, channel, url, title, text, fallback, repository, author, branch, commit string) SlackMessage {

  fields := []SlackAttachmentField{
    {
      Title: "Repository",
      Value: repository,
      Short: false,
    },
    {
      Title: "Author",
      Value: author,
      Short: false,
    },
    {
      Title: "Branch",
      Value: fmt.Sprintf("`%s`", branch),
      Short: false,
    },
    {
      Title: "Commit",
      Value: fmt.Sprintf("`%s`", commit),
      Short: false,
    },
  }

  attachments := []SlackAttachment{
    {
      Color: color,
      TitleLink: url,
      Title: title,
      Text: text,
      Fallback: fallback,
      Fields: fields,
    },
  }

  slackMessage := SlackMessage{
    Channel: channel,
    Username: "kube-git",
    IconURL: "https://raw.githubusercontent.com/appwavelets/kube-git/master/icon.png",
    Attachments: attachments,
  }

  return slackMessage

}
