package generate

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qiniupkg.com/x/log.v7"
)

const RemoveBFrameOpt = "-vcodec copy -bf 0"

func CutVideo(ctx context.Context, input, output, from, to string, verbose bool, outputOpt string) error {
	_ = os.MkdirAll(filepath.Dir(output), 0755)
	if from == "" && to == "" {
		//不裁剪，直接将视频复制过去
		originVideo, err := os.Open(input)
		if err != nil {
			return err
		}
		defer originVideo.Close()
		fileReader := bufio.NewReader(originVideo)
		destVideo, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			return err
		}
		defer destVideo.Close()
		fileWriter := bufio.NewWriter(destVideo)
		_, err = io.Copy(fileWriter, fileReader)
		return err
	}

	outputOpts := "-vcodec copy"
	if outputOpt != "" {
		outputOpts = outputOpt
	}
	args := fmt.Sprintf("-y -i %s -ss %s -to %s -avoid_negative_ts make_non_negative -an %s %s", input, from, to, strings.TrimSpace(outputOpts), output)
	cmd := exec.CommandContext(ctx, "ffmpeg", strings.Split(args, " ")...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	outputReader := io.MultiReader(stderr, stdout)

	err = cmd.Start()
	if err != nil {
		return err
	}

	outputBs, err := ioutil.ReadAll(outputReader)
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		err = fmt.Errorf("%s %s\n%s", "ffmpeg", args, string(outputBs))
		log.Error(err.Error())
		return err
	}

	if verbose {
		log.Infof("%s %s\n%s", "ffmpeg", args, string(outputBs))
	}

	log.Infof("generated video %s", output)
	return nil
}
