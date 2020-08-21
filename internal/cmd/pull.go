package cmd

import (
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"git.supremind.info/product/visionmind/test/event_test/internal/pull"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "pull test cases from remote",
	RunE:  pullCases,
}

var pullCasesParams = struct {
	configFile string
	output     string
	atomanArgs string
}{}

func init() {
	pullCmd.Flags().StringVarP(&pullCasesParams.configFile, "config", "", "conf/test.conf", "config file")
	pullCmd.Flags().StringVarP(&pullCasesParams.output, "output", "", ".", "Specify cases output directory")
	pullCmd.Flags().StringVarP(&pullCasesParams.atomanArgs, "args", "", "", "additional args pass to atoman")
	rootCmd.AddCommand(pullCmd)
}
func pullCases(cmd *cobra.Command, args []string) (err error) {
	config := com.ParseConfigFile(pullCasesParams.configFile)
	pull.PullCases(config, pullCasesParams.output, pullCasesParams.atomanArgs)
	return nil
}
