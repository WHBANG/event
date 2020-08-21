package pull

import (
	"io/ioutil"
	"os/exec"
	"strings"

	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"qiniupkg.com/x/log.v7"
)

func PullCases(config *com.Config, output string, args string) {
	argSlice := strings.Split(args, " ")
	atoman_args := append([]string{"pull", "volume", config.CasesVolume, "-p", config.CasesPrefix, "-o", output}, argSlice...)
	atoman := exec.Command("atoman", atoman_args...)

	//跳过atoman的自动更新
	atoman.Stdin = strings.NewReader("no")
	log.Println("using atoman to pull cases")
	stdout, err := atoman.StdoutPipe()
	if err != nil {
		log.Fatal(err.Error())
	}
	stderr, err := atoman.StderrPipe()
	if err != nil {
		log.Fatal(err.Error())
	}
	err = atoman.Start()
	if err != nil {
		log.Fatal(err.Error())
	}

	outBytes, _ := ioutil.ReadAll(stdout)
	errBytes, _ := ioutil.ReadAll(stderr)
	err = atoman.Wait()

	//atoman不论成功与否返回码都是0
	//按stdout是否有输出判断是否成功
	if err != nil || len(outBytes) == 0 {
		log.Println(string(errBytes))
		log.Fatal("failed to pull cases")
	} else {
		log.Println(string(outBytes))
		log.Println("pull cases successfully")
	}
}
