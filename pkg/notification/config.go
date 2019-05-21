package notification

import(
  "fmt"
  "github.com/ghodss/yaml"
	"io/ioutil"
  "strings"
)

type Config struct {
  Github map[string]GithubConfig `json:"github"`
  Slack  map[string]SlackConfig  `json:"slack"`
}

type GithubConfig struct {
  URL   string `json:"url"`
  Token string `json:"token"`
  API   string `json:"api"`
}

type SlackConfig struct {
  URL        string `json:"url"`
  WebhookURL string `json:"webhookURL"`
  Channel    string `json:"channel"`
}

func LoadConfig(configFile string) (*Config, error) {

  data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

  for k, v := range cfg.Github {
    if v.URL == "" {
      return nil, fmt.Errorf("Missing notification URL for GitHub notification config %s", k)
    }
    if v.Token == "" {
      return nil, fmt.Errorf("Missing token for GitHub notification config %s", k)
    }
  }

  for k, v := range cfg.Slack {
    if v.URL == "" {
      return nil, fmt.Errorf("Missing notification URL for Slack notification config %s", k)
    }
    if v.WebhookURL == "" {
      return nil, fmt.Errorf("Missing webhook URL for Slack notification config %s", k)
    }
  }

	return &cfg, nil

}

func (c Config) Notify(status string, kind string, namespace string, name string, annotations map[string]string) {

  if slackValue, slack := annotations["kubegit.appwavelets.com/slack"]; slack {
    if s, ok := c.Slack[slackValue]; ok {
      s.Notify(status, kind, namespace, name, annotations)
    }
  }

  if githubValue, github := annotations["kubegit.appwavelets.com/github"]; github {
    if g, ok := c.Github[githubValue]; ok {
      g.Notify(status, kind, namespace, name, annotations)
    }
  }

}

func renderURL(url string, resourceNamespace string, resourceName string) string {
  url = strings.Replace(url, "${NAMESPACE}", resourceNamespace, -1)
  url = strings.Replace(url, "${NAME}", resourceName, -1)
  return url
}
