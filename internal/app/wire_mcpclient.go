package app

import (
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/mcpclient"
)

var mcpClients sync.Map // *App → *mcpclient.Service

// MCPClient is the assistant's remote MCP servers (B5). Servers on private
// addresses follow providers.allow_private_networks; OAuth sends people
// back to the instance's public URL.
func (a *App) MCPClient() *mcpclient.Service {
	if v, ok := mcpClients.Load(a); ok {
		return v.(*mcpclient.Service)
	}
	s := &mcpclient.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log,
		AllowPrivate: func() bool { return a.Cfg != nil && a.Cfg.Providers.AllowPrivateNetworks },
		PublicURL:    a.PublicURL.Get}
	v, _ := mcpClients.LoadOrStore(a, s)
	return v.(*mcpclient.Service)
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		mcpclient.Register(h, a.MCPClient())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		river.AddWorker(a.JobReg.Workers, &mcpclient.PurgeWorker{S: a.MCPClient()})
		a.JobReg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return mcpclient.PurgeArgs{}, nil }, true)
	})
	closeHooks = append(closeHooks, func(a *App) { mcpClients.Delete(a) })
}
