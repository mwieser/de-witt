package config

type Auth struct {
	Username string `yaml:"username"`
	APIToken string `yaml:"apiToken"`
}

type ExternalJira struct {
	Name     string           `yaml:"name"`
	JiraURL  string           `yaml:"jiraURL"`
	Projects []ProjectMapping `yaml:"projects"`
	Epics    []Epic           `yaml:"epics"`
}

type ProjectMapping struct {
	ExternalProjectKey string `yaml:"externalProjectKey"`
	InternalIssueKey   string `yaml:"internalIssueKey"`
}

type Epic struct {
	ExternalEpicKey  string `yaml:"externalEpicKey"`
	InternalIssueKey string `yaml:"internalIssueKey"`
}

type AppConfig struct {
	Auth             Auth           `yaml:"auth"`
	InternalJiraURL  string         `yaml:"internalJiraURL"`
	External         []ExternalJira `yaml:"external"`
	Debug            bool           `yaml:"debug"`
	WorklogsPerIssue int            `yaml:"worklogsPerIssue"`
}
