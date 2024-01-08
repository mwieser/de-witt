package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"allaboutapps.dev/aw/de-witt/internal/config"
	"allaboutapps.dev/aw/de-witt/internal/util"
	jira "github.com/andygrunwald/go-jira"
	"github.com/rs/zerolog/log"
)

type mappingKey struct {
	JiraURL string
	Key     string
}

type Service struct {
	client *http.Client
	config config.AppConfig

	// mapps combination of jiraURL and projectKey to internal issueKey
	projectMapping map[mappingKey]string

	// maps combination of jiraURL and epicKey to internal issueKey
	epicMapping map[mappingKey]string
}

func NewService(config config.AppConfig) (*Service, error) {
	if config.Auth.Username == "" {
		return nil, fmt.Errorf("username is empty")
	}

	if config.Auth.APIToken == "" {
		return nil, fmt.Errorf("apiToken is empty")
	}

	if config.InternalJiraURL == "" {
		return nil, fmt.Errorf("internalJiraURL is empty")
	}

	tp := jira.BasicAuthTransport{
		Username: config.Auth.Username,
		Password: config.Auth.APIToken,
	}

	s := &Service{
		client: tp.Client(),
		config: config,
	}

	if err := s.initWorklogMapping(); err != nil {
		log.Err(err).Msg("failed to init worklog mapping")
		return nil, err
	}

	return s, nil
}

func (s *Service) initWorklogMapping() error {

	s.projectMapping = make(map[mappingKey]string)
	s.epicMapping = make(map[mappingKey]string)

	for i, external := range s.config.External {
		if external.JiraURL == "" {
			return fmt.Errorf("jiraURL is empty for project %v item %d", external.Name, i)
		}

		for i, project := range external.Projects {
			if project.ExternalProjectKey == "" {
				return fmt.Errorf("projects[%d].externalProjectKey is empty for project %v", i, external.Name)
			}

			if project.InternalIssueKey == "" {
				return fmt.Errorf("projects[%d].internalIssueKey is empty for project %v", i, external.Name)
			}

			key := mappingKey{
				JiraURL: external.JiraURL,
				Key:     project.ExternalProjectKey,
			}

			s.projectMapping[key] = project.InternalIssueKey
		}

		for _, epic := range external.Epics {
			if epic.ExternalEpicKey == "" {
				return fmt.Errorf("epics[%d].externalEpicKey is empty for project %v", i, external.Name)
			}

			if epic.InternalIssueKey == "" {
				return fmt.Errorf("epics[%d].internalIssueKey is empty for project %v", i, external.Name)
			}

			key := mappingKey{
				JiraURL: external.JiraURL,
				Key:     epic.ExternalEpicKey,
			}

			s.epicMapping[key] = epic.InternalIssueKey
		}
	}

	return nil
}

func (s *Service) BookInternal(date time.Time) error {
	globalLogger := log.With().Logger()
	log := log.With().Time("date", date).Logger()

	worklogs, err := s.GetWorkslogsByDate(date)
	if err != nil {
		log.Error().Err(err).Msg("failed to get worklogs by date")
		return err
	}

	jiraClient, err := jira.NewClient(s.client, s.config.InternalJiraURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to create internal jira client")
		return err
	}

	for _, worklog := range worklogs {
		log := log.With().Str("externalIssueKey", worklog.IssueKey).Logger()

		internalIssueKey, err := s.getInternalIssueKey(worklog)
		if err != nil {
			log.Debug().Err(err).Msg("failed to get internal issue key")
			return err
		}

		if !s.config.Debug {
			if err := s.bookInternalIssue(jiraClient, internalIssueKey, worklog); err != nil {
				log.Debug().Err(err).Msg("failed to book internal issue")
				return err
			}
		}

		globalLogger.Info().Str("issue", internalIssueKey).Str("timeSpent", worklog.TimeSpent).Msg("")
	}

	return nil
}

func (s *Service) bookInternalIssue(jiraClient *jira.Client, internalIssueKey string, worklog WorklogRecord) error {
	log := log.With().Str("internalIssueKey", internalIssueKey).Str("externalIssueKey", worklog.IssueKey).Str("timeSpent", worklog.TimeSpent).Logger()

	// check if worklog already exists
	worklogs, _, err := jiraClient.Issue.GetWorklogs(internalIssueKey)
	if err != nil {
		log.Err(err).Msg("failed to get worklogs")
		return err
	}

	for _, existing := range worklogs.Worklogs {
		if existing.Author.EmailAddress != s.config.Auth.Username {
			// skip worklogs from other users
			continue
		}

		if existing.Started == nil {
			// skip worklogs without started date
			continue
		}

		started := (time.Time)(*existing.Started)
		// check if worklog already exists with similar start time
		if worklog.Started.Equal(started) {
			log.Debug().Msg("worklog already exists")

			if existing.TimeSpentSeconds != worklog.TimeSpentSeconds {
				// update worklog if time spent is different
				worklogRequest := PostWorklogRequest{
					Started:   worklog.Started.Format("2006-01-02T15:04:05.000-0700"),
					TimeSpent: worklog.TimeSpent,
				}
				if worklog.Comment != "" {
					worklogRequest.Comment = buildWorklogComment(worklog.Comment)
				}

				req, err := jiraClient.NewRequest("PUT", "/rest/api/3/issue/"+internalIssueKey+"/worklog/"+existing.ID, worklogRequest)
				if err != nil {
					log.Err(err).Msg("failed to create update request")
					return err
				}
				response, err := jiraClient.Do(req, nil)
				if err != nil {
					bodyRaw, _ := io.ReadAll(response.Body)
					log.Debug().Str("body", string(bodyRaw)).Err(err).Msg("failed to update worklog")

					log.Err(err).Msg("failed to update worklog")
					return err
				}

				if response.StatusCode != http.StatusOK {
					log.Err(err).Msg("failed to update worklog, status not OK")
					return fmt.Errorf("failed to update worklog, status not OK")
				}

			}

			return nil
		}
	}

	worklogRequest := PostWorklogRequest{
		Started:   worklog.Started.Format("2006-01-02T15:04:05.000-0700"),
		TimeSpent: worklog.TimeSpent,
	}
	if worklog.Comment != "" {
		worklogRequest.Comment = buildWorklogComment(worklog.Comment)
	}

	req, err := jiraClient.NewRequest("POST", "/rest/api/3/issue/"+internalIssueKey+"/worklog", worklogRequest)
	if err != nil {
		log.Err(err).Msg("failed to create request")
		return err
	}
	response, err := jiraClient.Do(req, nil)
	if err != nil {
		log.Err(err).Msg("failed to post worklog")
		return err
	}

	if response.StatusCode != http.StatusCreated {
		log.Err(err).Msg("failed to post worklog, status not created")
		return fmt.Errorf("failed to post worklog, status not created")
	}

	return nil
}

func (s *Service) getInternalIssueKey(worklog WorklogRecord) (string, error) {

	key := mappingKey{
		JiraURL: worklog.JiraURL,
		Key:     worklog.ProjectKey,
	}
	issueKey, ok := s.projectMapping[key]
	if ok {
		return issueKey, nil
	}

	key = mappingKey{
		JiraURL: worklog.JiraURL,
		Key:     worklog.ParrentKey,
	}
	issueKey, ok = s.epicMapping[key]
	if ok {
		return issueKey, nil
	}

	return "", fmt.Errorf("Config missing for issue %s %s", worklog.IssueKey, worklog.JiraURL)
}

func (s *Service) GetWorkslogsByDate(date time.Time) ([]WorklogRecord, error) {
	worklogs := make([]WorklogRecord, 0)

	for _, external := range s.config.External {
		w, err := s.getWorkslogsByDateForJira(external.JiraURL, date)
		if err != nil {
			return nil, err
		}

		worklogs = append(worklogs, w...)
	}

	return worklogs, nil
}

func (s *Service) getWorkslogsByDateForJira(jiraURL string, date time.Time) ([]WorklogRecord, error) {
	log := log.With().Time("date", date).Str("jiraURL", jiraURL).Logger()
	worklogs := make([]WorklogRecord, 0)
	dateString := date.Format("2006-01-02")

	issues, err := s.getIssuesWithWorklogByDate(jiraURL, date)
	if err != nil {
		log.Debug().Err(err).Msg("failed to get issues by date")
		return nil, err
	}

	for _, issue := range issues {
		for _, worklog := range issue.Fields.Worklog.Worklogs {
			if worklog.Author.EmailAddress != s.config.Auth.Username {
				// skip worklogs from other users
				continue
			}

			if worklog.Started == nil {
				// skip worklogs without started date
				continue
			}

			started := (time.Time)(*worklog.Started)
			if started.Format("2006-01-02") != dateString {
				// skip worklogs with different date
				continue
			}

			if issue.Fields.Project.Key == "" && issue.Fields.Parent == nil {
				return nil, fmt.Errorf("issue %v has no project and no parent", issue.Key)
			}

			worklogRecord := WorklogRecord{
				JiraURL:          jiraURL,
				WorklogID:        worklog.ID,
				IssueKey:         issue.Key,
				TimeSpent:        worklog.TimeSpent,
				TimeSpentSeconds: worklog.TimeSpentSeconds,
				ProjectKey:       issue.Fields.Project.Key,
				Started:          started,
				Comment:          worklog.Comment,
			}

			if issue.Fields.Parent != nil {
				worklogRecord.ParrentKey = issue.Fields.Parent.Key
			}

			worklogs = append(worklogs, worklogRecord)
		}
	}

	return worklogs, nil
}

func (s *Service) getIssuesWithWorklogByDate(jiraURL string, date time.Time) ([]jira.Issue, error) {
	log := log.With().Str("jiraURL", jiraURL).Time("date", date).Logger()

	jiraClient, err := jira.NewClient(s.client, jiraURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to create jira client")
		return nil, err
	}

	dateString := date.Format("2006-01-02")
	jql := "worklogDate = " + dateString + " AND worklogAuthor = currentUser()"
	issues, _, err := jiraClient.Issue.Search(jql, &jira.SearchOptions{
		Fields: []string{
			"project",
			"parent",
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to search for issues")
		return nil, err
	}

	// load worklogs for each issue the user worked on today
	for _, issue := range issues {
		log = log.With().Str("issue", issue.Key).Logger()

		worklogs, err := s.getWorklogs(jiraURL, issue.Key, util.StartOfDay(date))
		if err != nil {
			log.Debug().Err(err).Msg("failed to get worklogs")
			return nil, err
		}

		issue.Fields.Worklog = worklogs
	}

	return issues, nil
}

func (s *Service) getWorklogs(jiraURL string, issueKey string, startedAfter time.Time) (*jira.Worklog, error) {
	log := log.With().Str("jiraURL", jiraURL).Str("issueKey", issueKey).Time("startedAfter", startedAfter).Logger()

	jiraClient, err := jira.NewClient(s.client, jiraURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to create jira client")
		return nil, err
	}

	maxResults := s.config.WorklogsPerIssue
	if maxResults == 0 {
		maxResults = 100
	}

	req, err := jiraClient.NewRequest("GET", fmt.Sprintf("/rest/api/2/issue/%s/worklog?startedAfter=%d&maxResults=%d", issueKey, startedAfter.UnixMilli(), maxResults), nil)
	if err != nil {
		log.Err(err).Msg("failed to build load worklogs request")
		return nil, err
	}
	response, err := jiraClient.Do(req, nil)
	if err != nil {
		bodyRaw, _ := io.ReadAll(response.Body)
		log.Debug().Str("body", string(bodyRaw)).Err(err).Msg("failed to load worklog")

		log.Err(err).Msg("failed to load worklog")
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		log.Err(err).Msg("failed to load worklog, status not OK")
		return nil, fmt.Errorf("failed to load worklog, status not OK")
	}

	worklogs := new(jira.Worklog)
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(worklogs); err != nil {
		log.Err(err).Msg("failed to unmarshal worklogs")
		return nil, err
	}

	return worklogs, nil
}
