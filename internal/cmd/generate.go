package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"git.supremind.info/product/visionmind/test/event_test/internal/generate"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate cases",
	RunE:  genCases,
}

var generateParams = struct {
	generate.GenerateArg
	worker     int
	configFile string
}{}

func init() {
	generateCmd.Flags().StringVarP(&generateParams.configFile, "config", "c", "conf/test.conf", "config file")
	generateCmd.Flags().StringVarP(&generateParams.BasePath, "input", "i", "", "directory to generate cases")
	generateCmd.Flags().BoolVarP(&generateParams.Verbose, "verbose", "", false, "In verbose mode, print more detail logs")
	generateCmd.Flags().IntVarP(&generateParams.worker, "worker", "", com.DefaultWorker, "concurrent number of generate cases")
	generateCmd.Flags().BoolVarP(&generateParams.OverWrite, "overwrite", "", false, "overwrite existing case and video")
	generateCmd.Flags().BoolVarP(&generateParams.Upload, "upload", "", false, "upload generated video&snapshot")
	generateCmd.Flags().BoolVarP(&generateParams.CutOnly, "cutonly", "", false, "do not generate case,cut video only")
	generateCmd.Flags().StringVarP(&generateParams.Args, "args", "", "", "additional args pass to ffmpeg")
	generateCmd.Flags().BoolVarP(&generateParams.RemoveBFrame, "remove_bframe", "b", false, "remove B frames in video")
	generateCmd.Flags().StringVarP(&generateParams.RemoteBasePath, "remote_path", "", "", "remote storage base path for video")
	generateCmd.Flags().BoolVarP(&generateParams.PR, "pr", "", false, "create a PR for generated cases")


	generateCmd.MarkFlagRequired("input")
	rootCmd.AddCommand(generateCmd)
}

func genCases(cmd *cobra.Command, args []string) (err error) {
	conf := com.ParseConfigFile(generateParams.configFile)
	if generateParams.worker > 0 && generateParams.worker != com.DefaultWorker {
		conf.Worker = generateParams.worker
	}

	gen, err := generate.NewCaseGenerate(&generateParams.GenerateArg, conf)
	if err != nil {
		return
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT)
	go func() {
		<-sigChannel
		cancelFunc()
	}()
	err = gen.StartGenerate(ctx)
	return
}
