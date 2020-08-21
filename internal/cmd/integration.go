package cmd

import (
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"git.supremind.info/product/visionmind/test/event_test/internal/integration"
	"github.com/spf13/cobra"
)

var integrationCmd = &cobra.Command{
	Use:   "integration",
	Short: "integration testing",
	RunE:  integrationTest,
}

type integrationGlobalParams struct {
	CasesFilePath string
	ConfigFile    string
	Worker        int
	integration.TestArgs
}

var (
	integrationParams integrationGlobalParams
	integrationConfig *com.TestCommon
)

func init() {
	integrationCmd.PersistentFlags().StringVarP(&integrationParams.CasesFilePath, "cases", "", "", "test cases filepath,or output path for --pull")
	integrationCmd.PersistentFlags().BoolVarP(&integrationParams.Verbose, "verbose", "", false, "In verbose mode, print more detail logs")
	integrationCmd.PersistentFlags().StringVarP(&integrationParams.ConfigFile, "config", "", "conf/test.conf", "config file")
	integrationCmd.PersistentFlags().IntVarP(&integrationParams.Worker, "worker", "w", com.DefaultWorker, "concurrent number of process tasks")
	integrationCmd.PersistentFlags().BoolVarP(&integrationParams.CreateOnly, "createonly", "", false, "create tasks without starting it")
	integrationCmd.PersistentFlags().StringVarP(&integrationParams.Match, "match", "", "", "label matching expr,support ==,!=,&&,||,()")
	integrationCmd.PersistentFlags().StringVarP(&integrationParams.Region, "region", "", "", "value of field region for created tasks")
	integrationCmd.PersistentFlags().IntVarP(&integrationParams.MaxChannel, "maxchannel", "", 1000, "max channel for sub device")
	integrationCmd.PersistentFlags().BoolVarP(&integrationParams.Delete, "delete", "d", false, "delete tasks after test finished")
	integrationCmd.PersistentFlags().StringVarP(&integrationParams.AssetHost, "asset_host", "", "", "temporary host name for case assets")
	integrationCmd.MarkFlagRequired("cases")

	integrationCmd.PersistentPreRunE = integrationPreFun
	rootCmd.AddCommand(integrationCmd)
}

func integrationTest(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}

func integrationPreFun(cmd *cobra.Command, args []string) (err error) {
	integrationConfig, err = com.NewTestCommon(integrationParams.ConfigFile)
	if err != nil {
		return
	}

	if integrationParams.Worker > 0 && integrationParams.Worker != com.DefaultWorker {
		integrationConfig.Config.Worker = integrationParams.Worker
	}

	if integrationParams.MaxChannel <= 0 {
		integrationParams.MaxChannel = 1000
	}
	return
}
