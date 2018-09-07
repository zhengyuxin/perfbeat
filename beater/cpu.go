package beater

import (
	"os/exec"
	"github.com/elastic/beats/libbeat/logp"
)

func execTop(result chan string)  {
	top := exec.Command("top", "-bn2", "-d1")
	out, err := top.CombinedOutput()
	if err != nil {
		logp.Err("exec top ", err)
		result <- ""
		return
	}
	s := string(out)
	result <- s
}

