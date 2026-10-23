package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
)

// UpgradeCommand is how a docker compose install upgrades.
const UpgradeCommand = "cd /opt/reputecast && docker compose pull && docker compose up -d"

// release is the stored latest release (instance_settings.latest_release).
type release struct {
	Tag         string    `json:"tag"`
	Name        string    `json:"name,omitempty"`
	PublishedAt time.Time `json:"published_at"`
	URL         string    `json:"url,omitempty"`
	Notes       []string  `json:"notes,omitempty"`
	Note        string    `json:"note,omitempty"`
	// Source is "github", or "demo" for the sample data.
	Source string `json:"source,omitempty"`
}

// InstanceRelease is the latest ReputeCast release.
type InstanceRelease struct {
	Tag         string    `json:"tag"`
	Name        string    `json:"name,omitempty"`
	PublishedAt time.Time `json:"publishedAt"`
	URL         string    `json:"url"`
	// Notes are the changelog's bullet points; Note its closing remark.
	Notes []string `json:"notes"`
	Note  string   `json:"note,omitempty"`
	// Demo marks the sample release of the demo data.
	Demo bool `json:"demo"`
}

// InstanceVersion is Instance admin › Instance & updates › Version & updates.
type InstanceVersion struct {
	Installed string `json:"installed" example:"v1.4.2"`
	Commit    string `json:"commit,omitempty"`
	// BuiltAt is when the running binary was built (the release date of a
	// release build).
	BuiltAt *time.Time `json:"builtAt,omitempty"`
	// Release: the installed version is a tagged release (vX.Y.Z).
	Release bool `json:"release"`
	// CheckEnabled is the instance switch; CheckAllowed is false when the
	// server configuration (updates.check) turns the check off entirely.
	CheckEnabled    bool             `json:"checkEnabled"`
	CheckAllowed    bool             `json:"checkAllowed"`
	CheckedAt       *time.Time       `json:"checkedAt,omitempty"`
	CheckError      string           `json:"checkError,omitempty"`
	Latest          *InstanceRelease `json:"latest,omitempty"`
	UpdateAvailable bool             `json:"updateAvailable"`
	UpgradeCommand  string           `json:"upgradeCommand"`
	ReleasesURL     string           `json:"releasesUrl"`
}

func (s *Service) repo() string {
	if s.Cfg != nil && s.Cfg.Updates.Repo != "" {
		return s.Cfg.Updates.Repo
	}
	return "hostrepute/reputecast"
}

func (s *Service) checkAllowed() bool { return s.Cfg == nil || s.Cfg.Updates.Check }

// Version returns the installed version and the latest release.
func (s *Service) Version(ctx context.Context) (InstanceVersion, error) {
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return InstanceVersion{}, err
	}
	v := version.Get()
	out := InstanceVersion{Installed: v.Version, Commit: v.Commit, Release: semver(v.Version) != nil, CheckEnabled: st.UpdateCheck, CheckAllowed: s.checkAllowed(),
		CheckedAt: st.UpdateCheckedAt, CheckError: st.UpdateCheckError, UpgradeCommand: UpgradeCommand,
		ReleasesURL: "https://github.com/" + s.repo() + "/releases"}
	if out.CheckedAt == nil {
		out.CheckedAt = st.LatestReleaseAt
	}
	if t, err := time.Parse(time.RFC3339, v.Date); err == nil {
		out.BuiltAt = &t
	}
	if len(st.LatestRelease) > 0 && string(st.LatestRelease) != "null" {
		var r release
		if json.Unmarshal(st.LatestRelease, &r) == nil && r.Tag != "" {
			url := r.URL
			if url == "" {
				url = "https://github.com/" + s.repo() + "/releases/tag/" + r.Tag
			}
			notes := r.Notes
			if notes == nil {
				notes = []string{}
			}
			out.Latest = &InstanceRelease{Tag: r.Tag, Name: r.Name, PublishedAt: r.PublishedAt, URL: url, Notes: notes, Note: r.Note, Demo: r.Source == "demo"}
			out.UpdateAvailable = newer(r.Tag, v.Version)
		}
	}
	return out, nil
}

// InstanceUpdateCheckInput turns the release check on or off.
type InstanceUpdateCheckInput struct {
	Enabled bool `json:"enabled"`
}

// SetUpdateCheck turns the daily release check on or off. Turning it on
// queues a check right away.
func (s *Service) SetUpdateCheck(ctx context.Context, in InstanceUpdateCheckInput) (InstanceVersion, error) {
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return InstanceVersion{}, err
	}
	if st.UpdateCheck != in.Enabled {
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.InstanceAdminSetUpdateCheck(ctx, in.Enabled); err != nil {
				return err
			}
			if in.Enabled && s.checkAllowed() && s.Jobs != nil {
				if c := s.Jobs(); c != nil {
					if _, err := c.InsertTx(ctx, tx, ReleaseCheckArgs{}, nil); err != nil {
						return err
					}
				}
			}
			summary := keep("audit.updateOff", nil)
			if in.Enabled {
				summary = keep("audit.updateOn", nil)
			}
			return s.Audit.LogWith(ctx, q, audit.Event{Action: "instance.update_check_changed", TargetType: "instance", TargetName: "Update check",
				Before: map[string]any{"enabled": st.UpdateCheck}, After: map[string]any{"enabled": in.Enabled}, Metadata: audit.KeepText(nil, "summary", summary)})
		})
		if err != nil {
			return InstanceVersion{}, err
		}
	}
	return s.Version(ctx)
}

// CheckNow fetches the latest release immediately (when the check is on).
func (s *Service) CheckNow(ctx context.Context) (InstanceVersion, error) {
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return InstanceVersion{}, err
	}
	if !st.UpdateCheck || !s.checkAllowed() {
		return InstanceVersion{}, api.Conflict(say(ctx, "problem.updateOff", nil))
	}
	_ = s.checkRelease(ctx)
	return s.Version(ctx)
}

// checkRelease asks GitHub for the latest release: one plain GET with no
// identifying data (no version, no instance id), and stores the answer or
// why it failed.
func (s *Service) checkRelease(ctx context.Context) error {
	u := s.ReleasesURL
	if u == "" {
		u = "https://api.github.com/repos/" + s.repo() + "/releases/latest"
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	// GitHub rejects requests without a User-Agent; this one names nothing
	// but the product.
	req.Header.Set("User-Agent", "ReputeCast-update-check")
	resp, err := s.client().Do(req)
	if err != nil {
		return s.Q.InstanceAdminRecordReleaseError(ctx, "GitHub didn't answer: "+err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return s.Q.InstanceAdminRecordReleaseError(ctx, fmt.Sprintf("GitHub answered %d %s.", resp.StatusCode, http.StatusText(resp.StatusCode)))
	}
	var gh struct {
		TagName     string    `json:"tag_name"`
		Name        string    `json:"name"`
		Body        string    `json:"body"`
		HTMLURL     string    `json:"html_url"`
		PublishedAt time.Time `json:"published_at"`
		Draft       bool      `json:"draft"`
		Prerelease  bool      `json:"prerelease"`
	}
	if err := json.Unmarshal(body, &gh); err != nil || gh.TagName == "" {
		return s.Q.InstanceAdminRecordReleaseError(ctx, "GitHub's answer had no release.")
	}
	notes, note := changelog(gh.Body)
	r := release{Tag: gh.TagName, Name: gh.Name, PublishedAt: gh.PublishedAt, URL: gh.HTMLURL, Notes: notes, Note: note, Source: "github"}
	return s.Q.InstanceAdminRecordRelease(ctx, mustJSON(r))
}

var mdNoise = regexp.MustCompile("[*_`]+")

// changelog takes the bullet points of a release body (up to six) and its
// last plain line as the closing note.
func changelog(body string) ([]string, string) {
	notes := []string{}
	note := ""
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			if len(notes) < 6 {
				notes = append(notes, cleanMarkdown(line[2:]))
			}
		default:
			note = cleanMarkdown(line)
		}
	}
	return notes, note
}

var mdLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)

func cleanMarkdown(s string) string {
	s = mdLink.ReplaceAllString(s, "$1")
	s = mdNoise.ReplaceAllString(s, "")
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return strings.TrimSpace(s)
}

var semverRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$`)

func semver(v string) []int {
	m := semverRe.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return nil
	}
	out := make([]int, 3)
	for i := range 3 {
		out[i], _ = strconv.Atoi(m[i+1])
	}
	if m[4] != "" {
		out = append(out, -1) // a pre-release sorts before its release
	}
	return out
}

// newer reports whether tag is a later version than installed. Builds that
// aren't tagged releases ("dev") never show an update.
func newer(tag, installed string) bool {
	a, b := semver(tag), semver(installed)
	if a == nil || b == nil {
		return false
	}
	for i := range 3 {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return len(a) == 3 && len(b) == 4
}

// ReleaseCheckArgs is the daily release check.
type ReleaseCheckArgs struct{}

// Kind implements river.JobArgs.
func (ReleaseCheckArgs) Kind() string { return "instance.release_check" }

// InsertOpts keeps one check at a time on the maintenance queue.
func (ReleaseCheckArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

// ReleaseCheckWorker runs the check when both the server configuration and
// the instance switch allow it; otherwise it makes no request at all.
type ReleaseCheckWorker struct {
	river.WorkerDefaults[ReleaseCheckArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ReleaseCheckWorker) Work(ctx context.Context, _ *river.Job[ReleaseCheckArgs]) error {
	if !w.S.checkAllowed() {
		return nil
	}
	st, err := w.S.Q.GetInstanceSettings(ctx)
	if err != nil {
		return err
	}
	if !st.UpdateCheck {
		return nil
	}
	// Once a day at most, however often the server restarts.
	if st.UpdateCheckedAt != nil && w.S.now().Sub(*st.UpdateCheckedAt) < 20*time.Hour {
		return nil
	}
	return w.S.checkRelease(ctx)
}
