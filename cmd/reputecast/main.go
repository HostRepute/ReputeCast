// Command reputecast is the single ReputeCast binary: the web app and API,
// background workers, webhook ingest, installation and maintenance commands.
package main

import (
	"os"

	"github.com/hostrepute/reputecast/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
