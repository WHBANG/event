package cmd

import (
	"git.supremind.info/product/visionmind/test/event_test/internal/integration"
	"github.com/spf13/cobra"
)

var duchaCmd = &cobra.Command{
	Use:   "ducha",
	Short: "ducha integration testing",
	RunE:  duchaIntegrationTest,
}

func init() {
	integrationCmd.AddCommand(duchaCmd)
}

func duchaIntegrationTest(cmd *cobra.Command, args []string) (err error) {

	cases := integration.ParseCasesFile(integrationParams.CasesFilePath)
	tester := integration.NewDuchaTester(&integrationParams.TestArgs, integrationConfig)
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
