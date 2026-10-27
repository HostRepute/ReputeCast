package renders

import (
	"context"
	"embed"
	"errors"
	"net/http"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
)

// Problems and states read in the caller's app language (i18n.LangFrom).
// The messages are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "renders")

var (
	mUnknownService = msgs.M("problem.unknownService")
	mUnknownClient  = msgs.M("problem.unknownClient")
	mNotConnected   = msgs.M("problem.notConnected")
	mStillRunning   = msgs.M("problem.stillRunning")
	mRefused        = msgs.M("problem.refused")
	mUnreachable    = msgs.M("problem.unreachable")
	mFailed         = msgs.M("problem.failed")
	mGaveUp         = msgs.M("render.gaveUp")
	mDisconnected   = msgs.M("render.disconnected")
	mNoScreenshots  = msgs.M("render.noScreenshots")
)

func say(ctx context.Context, m i18n.Msg, args ...i18n.Args) string {
	var a i18n.Args
	if len(args) > 0 {
		a = args[0]
	}
	return m.Text(i18n.LangFrom(ctx), a)
}

func notConnected(ctx context.Context) *api.Problem {
	return api.Err(http.StatusConflict, "rendering_not_connected", say(ctx, mNotConnected))
}

// describe says why a call to the service failed.
func describe(ctx context.Context, service string, err error) string {
	args := i18n.Args{"service": serviceName(service)}
	var ce *callError
	if errors.As(err, &ce) {
		switch ce.kind {
		case refused:
			return say(ctx, mRefused, args)
		case failed:
			args["status"] = ce.status
			return say(ctx, mFailed, args)
		}
	}
	args["reason"] = netguard.Describe(ctx, err)
	return say(ctx, mUnreachable, args)
}

// problem is a failed call as an API problem: a refused key is the key
// field's fault (422), anything else the service's (502).
func problem(ctx context.Context, service string, err error, keyField string) error {
	var ce *callError
	if !errors.As(err, &ce) {
		return err
	}
	msg := describe(ctx, service, err)
	if ce.kind == refused {
		if keyField != "" {
			return api.Invalid(keyField, msg)
		}
		return api.Err(http.StatusUnprocessableEntity, "rendering_key_refused", msg)
	}
	return api.Err(http.StatusBadGateway, "rendering_unavailable", msg)
}
