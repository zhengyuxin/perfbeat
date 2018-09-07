package beater

import (
	"os/exec"
	"github.com/elastic/beats/libbeat/logp"
	"strings"
	"strconv"
	"regexp"
)

type IotopResult struct {
	DiskRead float64
	DiskWrite float64
	SwapIn float64
}

func execIOTOP(result chan string){
	top := exec.Command("iotop", "-akb", "-Pn10")
	out, err := top.CombinedOutput()
	if err != nil {
		logp.Err("exec iotop ", err)
		result <- ""
		return
	}
	s := string(out)
	result <- s
}

func sliceSum(inputSlice[]float64) float64{
	var sumResult float64

	for _, value := range inputSlice{
		sumResult += value
	}

	return sumResult
}

func CheckProcessIO(iotopResult string, pidList []int32) map[int32]IotopResult {
	s := iotopResult
	pidIoTopResult := make(map[int32]IotopResult)

	if len(strings.TrimSpace(s)) == 0{
		for _, pid:= range pidList {
			 pidIoTopResult[pid] = IotopResult{-1.0, -1.0, -1.0}
		}

		return pidIoTopResult
	}

	lines := strings.Split(s, "\n")
	iotopHeadPattern,err := regexp.Compile("Total DISK READ :")
	if err != nil {
		logp.Err("iotop head pattern ", err)
	}
	iotopHeadList := iotopHeadPattern.FindAllString(s, -1)
	if len(iotopHeadList) ==0{
		for _, pid:= range pidList {
			pidIoTopResult[pid] = IotopResult{-2.0, -2.0, -2.0}
		}

		return pidIoTopResult
	}

	recordTime := float64(len(iotopHeadList))

	for _, pid := range pidList {
		var pidDiskRead []float64
		var pidDiskWrite []float64
		var pidSwapIn []float64

		for _, line := range lines {
			line = strings.TrimSpace(line)
			pidStr := strconv.FormatInt(int64(pid), 10)
			if strings.HasPrefix(line, pidStr+" "){
				lineFields := strings.Split(line, " ")
				var newFields []string
				for _, value := range lineFields{
					if value == ""{
						continue
					}
					newFields = append(newFields, value)
				}

				if len(newFields) > 0{
					diskRead,err := strconv.ParseFloat(newFields[3], 32)
					if err != nil {
						logp.Err("parse disk read of iotop", err)
					}
					pidDiskRead = append(pidDiskRead, diskRead)

					diskWrite, err:= strconv.ParseFloat(newFields[5], 32)
					if err != nil {
						logp.Err("parse disk write of iotop", err)
					}
					pidDiskWrite = append(pidDiskWrite, diskWrite)

					swapIn, err := strconv.ParseFloat(newFields[7], 32)
					if err != nil {
						logp.Err("parse swapin of iotop", err)
					}
					pidSwapIn = append(pidSwapIn, swapIn)
				}

			}
		}
		avgDiskRead := float64(sliceSum(pidDiskRead)/recordTime)
		avgDiskWrite:= float64(sliceSum(pidDiskWrite)/recordTime)
		avgSwapIn := float64(sliceSum(pidSwapIn)/recordTime)

		pidIoTopResult[pid] = IotopResult{avgDiskRead, avgDiskWrite, avgSwapIn}
	}
	return pidIoTopResult
}
