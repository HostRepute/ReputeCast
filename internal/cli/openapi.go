package cli

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/app"
)

func newOpenAPICmd() *cobra.Command {
	var (
		yamlOut bool
		public  bool
		oas     string
	)
	cmd := &cobra.Command{
		Use:   "openapi",
		Short: "Print the OpenAPI document (the web app's API types; with --public, the client SDKs')",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if oas != "3.1" && oas != "3.0" {
				return fmt.Errorf("--openapi is 3.1 or 3.0, not %q", oas)
			}
			hapi := humachi.New(chi.NewRouter(), app.NewHumaConfig())
			app.RegisterOperations(hapi)
			doc := hapi.OpenAPI()
			if public {
				var err error
				if doc, err = app.PublicSpec(doc); err != nil {
					return err
				}
			}
			var (
				b   []byte
				err error
			)
			switch {
			case yamlOut && oas == "3.0":
				b, err = doc.DowngradeYAML()
			case yamlOut:
				b, err = doc.YAML()
			case oas == "3.0":
				var flat []byte
				if flat, err = doc.Downgrade(); err == nil {
					var out bytes.Buffer
					err = json.Indent(&out, flat, "", "  ")
					b = append(out.Bytes(), '\n')
				}
			default:
				if b, err = json.MarshalIndent(doc, "", "  "); err == nil {
					b = append(b, '\n')
				}
			}
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(b)
			return err
		},
	}
	cmd.Flags().BoolVar(&yamlOut, "yaml", false, "print YAML instead of JSON")
	cmd.Flags().BoolVar(&public, "public", false, "only the operations an API key may call, for the client SDKs (sdk/openapi)")
	cmd.Flags().StringVar(&oas, "openapi", "3.1", "OpenAPI version: 3.1, or 3.0 for generators that don't read 3.1")
	return cmd
}
