// Command fakeidp runs the test OpenID Connect issuer locally, so the SSO
// setup screens can be tried end to end without a real identity provider:
//
//	go run ./internal/sso/ssotest/fakeidp -addr 127.0.0.1:9555 \
//	  -claims '{"sub":"marcus-1","email":"marcus@acme.io","name":"Marcus Chen","amr":["pwd","mfa"]}'
//
// Point a provider's issuer URL at the printed issuer with client ID
// test-client and secret test-secret. POST a JSON object to /_claims to
// change the claims of the next sign-ins, or ?error=access_denied to /_error.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/hostrepute/reputecast/internal/sso/ssotest"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9555", "listen address (local only)")
	path := flag.String("path", "", "issuer path, like /tenant/v2.0")
	claims := flag.String("claims", `{"sub":"user-1","email":"person@example.com","email_verified":true,"name":"Test Person"}`, "claims of the ID token (JSON)")
	flag.Parse()

	var c map[string]any
	if err := json.Unmarshal([]byte(*claims), &c); err != nil {
		log.Fatalf("claims: %v", err)
	}
	idp, err := ssotest.NewAt(*addr, *path)
	if err != nil {
		log.Fatal(err)
	}
	defer idp.Close()
	idp.Set(c, "")
	fmt.Printf("issuer %s · client test-client · secret test-secret\n", idp.Issuer())
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
