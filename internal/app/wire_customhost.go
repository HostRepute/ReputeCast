package app

import (
	"sync"

	"github.com/hostrepute/reputecast/internal/customhost"
)

// hostsByApp keeps one customhost.Hosts per App: the answers it keeps are
// shared by link tracking, short links, push clicks and the pages.
var hostsByApp sync.Map // *App → *customhost.Hosts

// customHosts says whose host a request came to. Its answers last a
// minute of the wall clock, whatever the app's clock says.
func (a *App) customHosts() *customhost.Hosts {
	if v, ok := hostsByApp.Load(a); ok {
		return v.(*customhost.Hosts)
	}
	v, _ := hostsByApp.LoadOrStore(a, &customhost.Hosts{Q: a.Q, Instance: a.URLs.Base})
	return v.(*customhost.Hosts)
}
