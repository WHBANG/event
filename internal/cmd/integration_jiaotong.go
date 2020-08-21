package cmd

import (
	"path"

	"git.supremind.info/product/visionmind/test/event_test/internal/integration"
	"git.supremind.info/product/visionmind/test/event_test/internal/pull"
	"github.com/spf13/cobra"
)

var jiaotongCmd = &cobra.Command{
	Use:   "jiaotong",
	Short: "jiaotong integration testing",
	RunE:  jiaotongIntegrationTest,
}
var jiaotongParams = struct {
	Pull       bool
	AtomanArgs string
}{}

func init() {
	jiaotongCmd.Flags().BoolVarP(&jiaotongParams.Pull, "pull", "", false, "pull test cases from remote")
	jiaotongCmd.Flags().StringVarP(&jiaotongParams.AtomanArgs, "args", "", "", "additional args pass to atoman,using with --pull")

	integrationCmd.AddCommand(jiaotongCmd)
}

func jiaotongIntegrationTest(cmd *cobra.Command, args []string) (err error) {

	if jiaotongParams.Pull {
		pull.PullCases(&integrationConfig.Config, integrationParams.CasesFilePath, jiaotongParams.AtomanArgs)
		integrationParams.CasesFilePath = path.Join(integrationParams.CasesFilePath, integrationConfig.Config.CasesPrefix)
	}

	cases := integration.ParseCasesFile(integrationParams.CasesFilePath)

	tester := integration.NewJiaotongTester(&integrationParams.TestArgs, integrationConfig)
	integrationTest := &integration.Test{
		TestCommon: integrationConfig,
		TestArgs:   &integrationParams.TestArgs,
		Tester:     tester,
	}
	err = integrationTest.ProcessCases(cases)
	if err != nil {
		return
	}

	return
}
