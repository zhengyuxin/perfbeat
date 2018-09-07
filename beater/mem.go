package beater

import (
	"github.com/elastic/beats/libbeat/logp"
	"strings"
	"strconv"
	"os/exec"
)

type memResult struct {
	Swap float64
	USS float64
	PSS float64
	RSS float64
}

func execSmem(memResults chan string)  {
	//smemCommand := exec.Command("smem", " -tc \"pid maps pss rss uss vss swap\"")
	// result will be : PID User Command Swap USS PSS RSS
	smemCommand := exec.Command("smem", " -t")
	logp.Info("use following command to get mem info\n%v", smemCommand.Args)
	smemOutPut, err:= smemCommand.CombinedOutput()
	if err != nil {
		logp.Err("Err of exec smem", err)
		memResults <- ""
		return
	}

	memResults <- string(smemOutPut)
}

func CheckProcessMem(memResults string, pidList []int32) map[int32]memResult{
	s := memResults
	pidMemResults := make(map[int32]memResult)
	
	if 0 == len(strings.TrimSpace(s)) {
		for _, value := range pidList {
			pidMemResults[value] = memResult{-1.0,-1.0,-1.0,-1.0}
		}
	}
	
	lines := strings.Split(s, "\n")

	for _, pid := range pidList {

		var pss,rss,uss,swap float64
		for _, line := range lines {
			pidStr := strconv.FormatInt(int64(pid), 10)
			if strings.HasPrefix(line, pidStr+" "){
				lineSlice := strings.Split(line, " ")
				var newSlice []string
				for _, value := range lineSlice {
					if value != ""{
						newSlice = append(newSlice, value)
					}
				}
				logp.Info("newSlice: %v", newSlice)
				swap,_ = strconv.ParseFloat(newSlice[4], 32)
				uss, _ = strconv.ParseFloat(newSlice[5], 32)
				pss, _ = strconv.ParseFloat(newSlice[6], 32)
				rss, _ = strconv.ParseFloat(newSlice[7], 32)
			}
		}
		pidMemResults[pid] = memResult{swap*1024, uss*1024, pss*1024,  rss*1024}
	}

	return pidMemResults
}
