package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/zhengyuxin/perfbeat/config"
	"runtime"
	process2 "github.com/shirou/gopsutil/process"
	"strings"
	"regexp"
	"net/http"
	"encoding/json"
)

type Perfbeat struct {
	done   chan struct{}
	config config.Config
	client beat.Client
	lastIndexTime time.Time
}

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &Perfbeat{
		done:   make(chan struct{}),
		config: config,
	}
	return bt, nil
}

func reMapFromString(str string, pattern string) map[string]string{
	reg, err:=regexp.Compile(pattern)
	if err != nil {
		println(err)
	}
	reMap := mapSubexpNames(reg.FindStringSubmatch(str), reg.SubexpNames())
	return reMap
}
func mapSubexpNames(m, n []string) map[string]string {
	m, n = m[1:], n[1:]
	r := make(map[string]string, len(m))
	for i, _ := range n {
		r[n[i]] = m[i]
	}
	return r
}

func getJson(url string, target interface{}) error{
	resp, err:= http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(target)
}

func (bt *Perfbeat) Run(b *beat.Beat) error {
	runtime.GOMAXPROCS(runtime.NumCPU())

	//cpuresult := make(chan string)
	memresult := make(chan string)
	ioresult := make(chan string)
	//netresult := make(chan string)
	pattern := "-uuid (?P<uuid>[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}){1}"

	logp.Info("perfbeat is running! Hit CTRL-C to stop it.")

	var err error

	logp.Info("Try to connect Publisher pipeline")
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	logp.Info("get New Ticker")
	ticker := time.NewTicker(bt.config.Period)
	counter := 1
	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		processList, err:=process2.Processes()
		if err != nil {
			logp.Err("Error when trying get list of pids: %v", err)
		}

		var pidList []int32

		for _, value := range processList {
			pidCommand,_ := value.Cmdline()
			pidCommand = string(pidCommand)
			if strings.Contains(pidCommand, "qemu-system-x86_64") {
				pidList = append(pidList, value.Pid)
			}
		}

		go execIOTOP(ioresult)
		go execSmem(memresult)

		iotopOutput := <- ioresult
		smemOutput := <- memresult
		pidMemResults:= CheckProcessMem(smemOutput, pidList)
		pidIoTopResult:= CheckProcessIO(iotopOutput, pidList)
		logp.Info("pidList:%v",pidList )
		logp.Info("pidMemResults:%v",pidMemResults )
		logp.Info("pidIoTopResult:%v",pidIoTopResult )

		for _, value := range processList {
			pidCommand,_ := value.Cmdline()
			pidCommand = string(pidCommand)
			if strings.Contains(pidCommand, "qemu-system-x86_64") {
				//logp.Info("%v", pidCommand)

				//go execTop(cpuresult)



				memoryPct32, _ := value.MemoryPercent()
				memoryPct := float64(memoryPct32)


				cpuPct, _ := value.CPUPercent()
				uuid_map:=reMapFromString(pidCommand, pattern)

				if err != nil {
					logp.Err("error when try to get memory info", err)
				}
				event := beat.Event{
					Timestamp: time.Now(),
					Fields: common.MapStr{
						"type":    b.Info.Name,
						"counter": counter,
						"pid": value.Pid,
						//"command":pidCommand,
						"uuid":uuid_map["uuid"],
						"cpuPct": cpuPct,
						"memoryPct": memoryPct,
						"memory.pss":pidMemResults[value.Pid].PSS,
						"memory.rss":pidMemResults[value.Pid].RSS,
						"memory.uss":pidMemResults[value.Pid].USS,
						"memory.swap":pidMemResults[value.Pid].Swap,
						"disk.read":pidIoTopResult[value.Pid].DiskRead,
						"disk.write":pidIoTopResult[value.Pid].DiskWrite,
						"disk.swapin":pidIoTopResult[value.Pid].SwapIn,
					},
				}
				bt.client.Publish(event)
				logp.Info("Event sent")
				counter++
			}


			}
		}

	}


func (bt *Perfbeat) Stop() {
	bt.client.Close()
	close(bt.done)
}

