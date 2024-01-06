
# De Witt - Booker

## Overview
This command-line interface (CLI) tool, written in Go, is designed to map Jira time bookings from multiple external Jira instances to a single internal Jira instance. The mapping can be done via either project key or epic key. This tool is particularly useful for consolidating time tracking across various projects and epics from different Jira environments into one central location.

## Configuration
To use this tool, create a `config.yaml` file with the following structure:

```yaml
de-witt:
  auth:
    username: "<EMAIL>"
    apiToken: "<TOKEN>"
  internalJiraURL: "https://my-internal-jira.atlassian.net"
  external:
    - name: "test1"
      jiraURL: "https://my-test1-external-jira.atlassian.net"
      projects:
        - externalProjectKey: "TEST"
          internalIssueKey: "SOME-2"
    - name: "test2"
      jiraURL: "https://my-test2-external-jira.atlassian.net"
      epics:
        - externalEpicKey: "TEST2-12"
          internalIssueKey: "OTHER-1"
        - externalEpicKey: "TEST-23"
          internalIssueKey: "OTHER-2"
```

Replace `<EMAIL>` and `<TOKEN>` with your Jira credentials. Define your internal and external Jira instances along with the corresponding project and epic keys.

## Usage
1. Place the `config.yaml` file in the same directory as the tool.
2. Run the tool: `./de-witt`.

An optional location for the config file can be provided via the `config` flag.

```
Usage:
  app [flags]

Flags:
  -c, --config string   (Optional) config file path
  -h, --help            help for app
  -v, --version         version for app
```

The tool will read the configuration file and map the time bookings from the external Jira projects/epics to the specified issues in your internal Jira.

## Features
- Support for multiple external Jira instances.
- Mapping time entries based on project key or epic key.
- Create worklog with comments
- Update time spent and comment on worklogs with the same start date

## Contributions
Contributions to this project are welcome. Please follow the standard Git workflow - fork the repository, make your changes, and submit a pull request.

## License
This tool is licensed under MIT. Please see the LICENSE file for more details.