package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/keel-hq/keel/types"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

var newGithubWebhooksCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "Github_webhook_requests_total",
		Help: "How many /v1/webhooks/github requests processed, partitioned by image.",
	},
	[]string{"image"},
)

func init() {
	prometheus.MustRegister(newGithubWebhooksCounter)
}

type githubWebhook struct {
	Action          string `json:"action"`
	RegistryPackage struct {
		CreatedAt string `json:"created_at"`
		HTMLURL   string `json:"html_url"`
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Owner     struct {
			AvatarURL         string `json:"avatar_url"`
			EventsURL         string `json:"events_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			GravatarID        string `json:"gravatar_id"`
			HTMLURL           string `json:"html_url"`
			ID                int    `json:"id"`
			Login             string `json:"login"`
			NodeID            string `json:"node_id"`
			OrganizationsURL  string `json:"organizations_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			ReposURL          string `json:"repos_url"`
			SiteAdmin         bool   `json:"site_admin"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			Type              string `json:"type"`
			URL               string `json:"url"`
		} `json:"owner"`
		PackageType    string `json:"package_type"`
		PackageVersion struct {
			Author struct {
				AvatarURL         string `json:"avatar_url"`
				EventsURL         string `json:"events_url"`
				FollowersURL      string `json:"followers_url"`
				FollowingURL      string `json:"following_url"`
				GistsURL          string `json:"gists_url"`
				GravatarID        string `json:"gravatar_id"`
				HTMLURL           string `json:"html_url"`
				ID                int    `json:"id"`
				Login             string `json:"login"`
				NodeID            string `json:"node_id"`
				OrganizationsURL  string `json:"organizations_url"`
				ReceivedEventsURL string `json:"received_events_url"`
				ReposURL          string `json:"repos_url"`
				SiteAdmin         bool   `json:"site_admin"`
				StarredURL        string `json:"starred_url"`
				SubscriptionsURL  string `json:"subscriptions_url"`
				Type              string `json:"type"`
				URL               string `json:"url"`
			} `json:"author"`
			Body                string        `json:"body"`
			BodyHTML            string        `json:"body_html"`
			CreatedAt           string        `json:"created_at"`
			HTMLURL             string        `json:"html_url"`
			ID                  int           `json:"id"`
			InstallationCommand string        `json:"installation_command"`
			Manifest            string        `json:"manifest"`
			Metadata            []interface{} `json:"metadata"`
			PackageFiles        []struct {
				ContentType string      `json:"content_type"`
				CreatedAt   string      `json:"created_at"`
				DownloadURL string      `json:"download_url"`
				ID          int         `json:"id"`
				Md5         interface{} `json:"md5"`
				Name        string      `json:"name"`
				Sha1        interface{} `json:"sha1"`
				Sha256      string      `json:"sha256"`
				Size        int         `json:"size"`
				State       string      `json:"state"`
				UpdatedAt   string      `json:"updated_at"`
			} `json:"package_files"`
			Summary         string `json:"summary"`
			TargetCommitish string `json:"target_commitish"`
			TargetOid       string `json:"target_oid"`
			UpdatedAt       string `json:"updated_at"`
			Version         string `json:"version"`
		} `json:"package_version"`
		Registry struct {
			AboutURL string `json:"about_url"`
			Name     string `json:"name"`
			Type     string `json:"type"`
			URL      string `json:"url"`
			Vendor   string `json:"vendor"`
		} `json:"registry"`
		UpdatedAt string `json:"updated_at"`
	} `json:"registry_package"`
	Repository struct {
		ArchiveURL       string      `json:"archive_url"`
		Archived         bool        `json:"archived"`
		AssigneesURL     string      `json:"assignees_url"`
		BlobsURL         string      `json:"blobs_url"`
		BranchesURL      string      `json:"branches_url"`
		CloneURL         string      `json:"clone_url"`
		CollaboratorsURL string      `json:"collaborators_url"`
		CommentsURL      string      `json:"comments_url"`
		CommitsURL       string      `json:"commits_url"`
		CompareURL       string      `json:"compare_url"`
		ContentsURL      string      `json:"contents_url"`
		ContributorsURL  string      `json:"contributors_url"`
		CreatedAt        string      `json:"created_at"`
		DefaultBranch    string      `json:"default_branch"`
		DeploymentsURL   string      `json:"deployments_url"`
		Description      string      `json:"description"`
		Disabled         bool        `json:"disabled"`
		DownloadsURL     string      `json:"downloads_url"`
		EventsURL        string      `json:"events_url"`
		Fork             bool        `json:"fork"`
		Forks            int         `json:"forks"`
		ForksCount       int         `json:"forks_count"`
		ForksURL         string      `json:"forks_url"`
		FullName         string      `json:"full_name"`
		GitCommitsURL    string      `json:"git_commits_url"`
		GitRefsURL       string      `json:"git_refs_url"`
		GitTagsURL       string      `json:"git_tags_url"`
		GitURL           string      `json:"git_url"`
		HasDownloads     bool        `json:"has_downloads"`
		HasIssues        bool        `json:"has_issues"`
		HasPages         bool        `json:"has_pages"`
		HasProjects      bool        `json:"has_projects"`
		HasWiki          bool        `json:"has_wiki"`
		Homepage         string      `json:"homepage"`
		HooksURL         string      `json:"hooks_url"`
		HTMLURL          string      `json:"html_url"`
		ID               int         `json:"id"`
		IssueCommentURL  string      `json:"issue_comment_url"`
		IssueEventsURL   string      `json:"issue_events_url"`
		IssuesURL        string      `json:"issues_url"`
		KeysURL          string      `json:"keys_url"`
		LabelsURL        string      `json:"labels_url"`
		Language         string      `json:"language"`
		LanguagesURL     string      `json:"languages_url"`
		License          interface{} `json:"license"`
		MergesURL        string      `json:"merges_url"`
		MilestonesURL    string      `json:"milestones_url"`
		MirrorURL        interface{} `json:"mirror_url"`
		Name             string      `json:"name"`
		NodeID           string      `json:"node_id"`
		NotificationsURL string      `json:"notifications_url"`
		OpenIssues       int         `json:"open_issues"`
		OpenIssuesCount  int         `json:"open_issues_count"`
		Owner            struct {
			AvatarURL         string `json:"avatar_url"`
			EventsURL         string `json:"events_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			GravatarID        string `json:"gravatar_id"`
			HTMLURL           string `json:"html_url"`
			ID                int    `json:"id"`
			Login             string `json:"login"`
			NodeID            string `json:"node_id"`
			OrganizationsURL  string `json:"organizations_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			ReposURL          string `json:"repos_url"`
			SiteAdmin         bool   `json:"site_admin"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			Type              string `json:"type"`
			URL               string `json:"url"`
		} `json:"owner"`
		Private         bool   `json:"private"`
		PullsURL        string `json:"pulls_url"`
		PushedAt        string `json:"pushed_at"`
		ReleasesURL     string `json:"releases_url"`
		Size            int    `json:"size"`
		SSHURL          string `json:"ssh_url"`
		StargazersCount int    `json:"stargazers_count"`
		StargazersURL   string `json:"stargazers_url"`
		StatusesURL     string `json:"statuses_url"`
		SubscribersURL  string `json:"subscribers_url"`
		SubscriptionURL string `json:"subscription_url"`
		SvnURL          string `json:"svn_url"`
		TagsURL         string `json:"tags_url"`
		TeamsURL        string `json:"teams_url"`
		TreesURL        string `json:"trees_url"`
		UpdatedAt       string `json:"updated_at"`
		URL             string `json:"url"`
		Watchers        int    `json:"watchers"`
		WatchersCount   int    `json:"watchers_count"`
	} `json:"repository"`
	Sender struct {
		AvatarURL         string `json:"avatar_url"`
		EventsURL         string `json:"events_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		GravatarID        string `json:"gravatar_id"`
		HTMLURL           string `json:"html_url"`
		ID                int    `json:"id"`
		Login             string `json:"login"`
		NodeID            string `json:"node_id"`
		OrganizationsURL  string `json:"organizations_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		ReposURL          string `json:"repos_url"`
		SiteAdmin         bool   `json:"site_admin"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		Type              string `json:"type"`
		URL               string `json:"url"`
	} `json:"sender"`
}

// githubHandler - used to react to github webhooks
func (s *TriggerServer) githubHandler(resp http.ResponseWriter, req *http.Request) {
	gw := githubWebhook{}
	if err := json.NewDecoder(req.Body).Decode(&gw); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.githubHandler: failed to decode request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if gw.RegistryPackage.PackageType != "docker" {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "registry package type was not docker")
	}

	if gw.Repository.FullName == "" { // github package name
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "repository full name cannot be empty")
		return
	}

	if gw.RegistryPackage.Name == "" { // github package name
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "repository package name cannot be empty")
		return
	}

	if gw.RegistryPackage.PackageVersion.Version == "" { // tag
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "repository tag cannot be empty")
		return
	}

	event := types.Event{}
	event.CreatedAt = time.Now()
	event.TriggerName = "github"
	event.Repository.Name = strings.Join(
		[]string{"docker.pkg.github.com", gw.Repository.FullName, gw.RegistryPackage.Name},
		"/",
	)
	event.Repository.Tag = gw.RegistryPackage.PackageVersion.Version

	s.trigger(event)

	resp.WriteHeader(http.StatusOK)

	newGithubWebhooksCounter.With(prometheus.Labels{"image": event.Repository.Name}).Inc()
}
