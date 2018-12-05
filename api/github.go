package api

import (
	"bufio"
	"net/http"
	"strings"

	"github.com/google/go-github/v19/github"
)

const (
	newIssue = "opened"
)

// issueType can be presentaion or request
type issueType string

var (
	presentationText = "i'dliketoshowsomething"

	requestText = "i'dliketolearnsomething"

	request      issueType = "request"
	presentation issueType = "presentation"
	unknown      issueType = "unknown"
)

func handleGithubIssueTrigger(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte(cfg.GithubWebhookRepoAgendaSecretKey))
	if err != nil {
		logger.Fatal(err)
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		logger.Fatal(err)
	}

	switch eT := event.(type) {
	case *github.IssuesEvent:
		processIssuesEvent(eT)
	default:
		logger.Info("Info not github issue")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// processIssuesEventProcess the github IssuesEvent
// for various actions.
func processIssuesEvent(event *github.IssuesEvent) {

	// action has to be "opened" to take action.
	action := event.GetAction()
	if action != newIssue {
		logger.Info("Not a new opened issue event")
		return
	}

	issue := event.GetIssue()

	// we are only interested in github issue for this trigger.
	// and issue should not be pull request too, as every pull
	// request is also an issue in github api
	if issue.IsPullRequest() == true {
		return
	}

	// check if labels if already present.
	// if present no need to assign a new label.
	var labels []string
	for _, label := range issue.Labels {
		name := label.GetName()
		labels = append(labels, name)
	}

	if len(labels) != 0 {
		return
	}

	// get the body of the issue.
	issueBody := issue.GetBody()
	scanner := bufio.NewScanner(strings.NewReader(issueBody))

	var firstLine string
	for scanner.Scan() {
		firstLine = scanner.Text()
		// break at first non empty text
		if firstLine != "" {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Error("reading input: " + err.Error())
	}

	issueIs := isLearnOrRequest(firstLine)
	logger.Info(issueIs)

}

func isLearnOrRequest(line string) issueType {
	// check if first line of text is equal to -
	// presentation topic #I'd Like To Show Something.
	// or requested topic #I'd Like To Learn Something.

	// trim all space
	text := strings.TrimSpace(line)
	// remove all # character ## or ### can be different
	// better check for occurence of words.
	text = strings.Trim(text, "#")
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Split(bufio.ScanWords)

	var sb strings.Builder
	for scanner.Scan() {
		word := scanner.Text()
		word = strings.ToLower(word)
		sb.WriteString(word)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("reading input: " + err.Error())
	}

	sbString := sb.String()
	if strings.Contains(sbString, requestText) {
		return request
	}
	if strings.Contains(sbString, presentationText) {
		return presentation
	}

	return unknown
}
