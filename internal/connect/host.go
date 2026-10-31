package connect

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Where ReputeCast runs decides the SES access it recommends (review
// PR-5): on AWS (EC2, ECS, EKS, Lambda) the CloudFormation stack's role
// needs no long-lived keys, because the instance's own role assumes it;
// anywhere else assuming that role still needs base AWS credentials on
// this server, so IAM access keys are simpler and recommended.

// HostInfo is what ReputeCast found out about its own host.
type HostInfo struct {
	// OnAWS: instance or task metadata answered, or the runtime said so.
	OnAWS bool
	// Kind is ec2, ecs, eks or lambda on AWS, empty elsewhere.
	Kind string
	// BaseCredentials: this server has AWS credentials of its own to
	// assume a role with (environment, shared files, or the instance's).
	BaseCredentials bool
	CheckedAt       time.Time
}

// RecommendedAccess is the SES access to suggest on this host.
func (h HostInfo) RecommendedAccess() string {
	if h.OnAWS {
		return "role"
	}
	return "keys"
}

// HostDetector finds out where this process runs; the answer is kept for
// ten minutes. Probe overrides the metadata check (tests).
type HostDetector struct {
	Probe func(ctx context.Context) bool
	Env   func(string) string
	Now   func() time.Time

	mu     sync.Mutex
	cached *HostInfo
}

// hostTTL is how long a host check is kept.
const hostTTL = 10 * time.Minute

// Detect returns where ReputeCast runs.
func (d *HostDetector) Detect(ctx context.Context) HostInfo {
	now := time.Now
	if d.Now != nil {
		now = d.Now
	}
	d.mu.Lock()
	if d.cached != nil && now().Sub(d.cached.CheckedAt) < hostTTL {
		h := *d.cached
		d.mu.Unlock()
		return h
	}
	d.mu.Unlock()
	getenv := os.Getenv
	if d.Env != nil {
		getenv = d.Env
	}
	h := HostInfo{CheckedAt: now()}
	switch {
	case getenv("AWS_LAMBDA_FUNCTION_NAME") != "":
		h.OnAWS, h.Kind = true, "lambda"
	case getenv("ECS_CONTAINER_METADATA_URI_V4") != "" || getenv("ECS_CONTAINER_METADATA_URI") != "" || getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI") != "":
		h.OnAWS, h.Kind = true, "ecs"
	case getenv("AWS_WEB_IDENTITY_TOKEN_FILE") != "" && getenv("AWS_ROLE_ARN") != "":
		h.OnAWS, h.Kind = true, "eks"
	default:
		probe := d.Probe
		if probe == nil {
			probe = ec2MetadataAnswers
		}
		if probe(ctx) {
			h.OnAWS, h.Kind = true, "ec2"
		}
	}
	h.BaseCredentials = h.OnAWS || getenv("AWS_ACCESS_KEY_ID") != "" || getenv("AWS_PROFILE") != "" || sharedCredentials(getenv)
	d.mu.Lock()
	d.cached = &h
	d.mu.Unlock()
	return h
}

// sharedCredentials reports whether the AWS shared credentials file exists.
func sharedCredentials(getenv func(string) string) bool {
	path := getenv("AWS_SHARED_CREDENTIALS_FILE")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		path = filepath.Join(home, ".aws", "credentials")
	}
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

// ec2MetadataAnswers asks the EC2 instance metadata service for a session
// token (IMDSv2). The address is fixed and link-local, never taken from
// input, and the request never goes through a proxy; off EC2 nothing
// answers within the short timeout.
func ec2MetadataAnswers(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 600*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://169.254.169.254/latest/api/token", nil)
	if err != nil {
		return false
	}
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "30")
	c := &http.Client{
		Transport: &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 400 * time.Millisecond}).DialContext, DisableKeepAlives: true},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := c.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
