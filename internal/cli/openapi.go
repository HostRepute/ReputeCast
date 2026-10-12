package cli

import (
	"encoding/json"
	"fmt"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/app"
)

func newOpenAPICmd() *cobra.Command {
	var yamlOut bool
	cmd := &cobra.Command{
		Use:   "openapi",
		Short: "Print the OpenAPI 3.1 document (used to generate the web app's API types)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			hapi := humachi.New(chi.NewRouter(), app.NewHumaConfig())
			app.RegisterOperations(hapi)
			if yamlOut {
				b, err := hapi.OpenAPI().YAML()
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(b)
				return err
			}
			b, err := json.MarshalIndent(hapi.OpenAPI(), "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return err
		},
	}
	cmd.Flags().BoolVar(&yamlOut, "yaml", false, "print YAML instead of JSON")
	return cmd
}
