package cmd

import (
	"fmt"
	"os"

	server "github.com/simple-apiserver/pkg"
	"github.com/spf13/cobra"
)

var port string

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().StringVarP(&port, "port", "p", "8082", "This flag sets the port of the server")
}

var rootCmd = &cobra.Command{
	Use:   ".",
	Short: "This is the main command",
	Long:  `This is the main command of simple apiserver`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Simple apiserver starts")
		server.SetFlags(port)
		server.StartServer()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Current version of the api server",
	Long:  "This command will print the current version of the simple apiserver",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("v0.0.1")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
