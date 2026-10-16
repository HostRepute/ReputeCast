// The integration apps are JavaScript and PHP projects with no Go code.
// This file makes integrations/ a module of its own, so the ReputeCast
// module's ./... (go build, go test, go vet, golangci-lint) never reaches
// their installed dependencies: some npm packages ship Go files.
module github.com/hostrepute/reputecast/integrations

go 1.27.2
