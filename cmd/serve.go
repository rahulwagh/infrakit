// cmd/serve.go
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/rahulwagh/infrakit/server" // CHANGE THIS
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts a local web server to search resources from a browser.",
	Run: func(cmd *cobra.Command, args []string) {
		// This simply calls the StartServer function from our server package.
		server.StartServer()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}