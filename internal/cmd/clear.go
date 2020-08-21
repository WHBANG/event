package cmd

import (
	"git.supremind.info/product/visionmind/test/event_test/internal/clear"
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "clear devices and tasks related to test",
	RunE:  clearTest,
}

var clearCmdParams = struct {
	verbose    bool
	configFile string
	taskType   string
}{}

func init() {
	clearCmd.Flags().BoolVarP(&clearCmdParams.verbose, "vervose", "", false, "In verbose mode, print more detail logs")
	clearCmd.Flags().StringVarP(&clearCmdParams.configFile, "config", "", "conf/test.conf", "config file")
	clearCmd.Flags().StringVarP(&clearCmdParams.taskType, "type", "", "", "type of tasks about to be cleared")

	rootCmd.AddCommand(clearCmd)
}

func clearTest(cmd *cobra.Command, args []string) (err error) {
	testCommon, err := com.NewTestCommon(clearCmdParams.configFile)
	if err != nil {
		return
	}

	c := &clear.Clear{
		TestCommon: *testCommon,
		TaskType:   clearCmdParams.taskType,
	}

	err = c.Do()
	return
}
