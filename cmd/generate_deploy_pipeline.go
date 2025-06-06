package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configFile string

// generateDeployPipelineCmd represents the generateDeployPipeline command
var generateDeployPipelineCmd = &cobra.Command{
	Use:   "generate-app-pipeline",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("generateDeployPipeline called")
	},
}

func init() {
	rootCmd.AddCommand(generateDeployPipelineCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	generateDeployPipelineCmd.Flags().StringVarP(&configFile, "config-file", "c", "", "deployment config file")

	//generateDeployPipelineCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// generateDeployPipelineCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
