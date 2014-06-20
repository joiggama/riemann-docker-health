package main

import (
	"flag"
	"fmt"
	"github.com/amir/raidman"
	"github.com/supherman/riemann-docker-health/docker"
	"github.com/supherman/riemann-docker-health/docker/cpu"
	"github.com/supherman/riemann-docker-health/docker/mem"
	"log"
	"os"
	"time"
)

var (
	cpuWarning  = flag.Int("cpu_warning", 50, "CPU warning threshold")
	cpuCritical = flag.Int("cpu_critical", 90, "CPU critical threshold")
	memWarning  = flag.Int("memory_warning", 50, "Memory warning threshold")
	memCritical = flag.Int("memory_critical", 90, "Memory critical threshold")
)

type Threshold struct {
	warning  int
	critical int
}

func Hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Println(err)
	}
	return hostname
}

func Alert(event *raidman.Event) {
	c, err := raidman.Dial("tcp", "localhost:5555")
	if err != nil {
		panic(err)
	}
	err = c.Send(event)

	if err != nil {
		panic(err)
	}
	c.Close()
}

func ComputeState(metric int, threshold *Threshold) string {
	switch {
	case metric >= threshold.critical:
		return "critical"
	case metric >= threshold.warning && metric < threshold.critical:
		return "warning"
	}
	return "ok"
}

func AlertCPU(container string, threshold *Threshold) {
	containerCpu, _ := cpu.GetUsage(container)

	metric := int(containerCpu)
	state := ComputeState(metric, threshold)

	var cpuEvent = &raidman.Event{
		State:   state,
		Service: "cpu",
		Metric:  metric,
		Ttl:     10,
		Host:    fmt.Sprintf("%s %s", Hostname(), container),
	}
	Alert(cpuEvent)
}

func AlertMemory(container string, threshold *Threshold) {
	containerMem, _ := mem.GetPercentage(container)

	metric := int(containerMem)
	state := ComputeState(metric, threshold)

	var memEvent = &raidman.Event{
		State:   state,
		Service: "memory",
		Metric:  metric,
		Ttl:     10,
		Host:    fmt.Sprintf("%s %s", Hostname(), container),
	}
	Alert(memEvent)
}

func main() {
	flag.Parse()
	cpuThreshold := Threshold{*cpuWarning, *cpuCritical}
	memThreshold := Threshold{*memWarning, *memCritical}

	go cpu.Monitor()

	tick := time.NewTicker(1 * time.Second)

	for {
		<-tick.C
		containers, err := docker.ListContainers()
		if err != nil {
			log.Println(err)
			continue
		}

		for _, container := range containers {
			AlertCPU(container, &cpuThreshold)
			AlertMemory(container, &memThreshold)
		}
	}
}
