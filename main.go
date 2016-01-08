package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

const configPath = "/etc/lagpipe.conf"

type Config struct {
	Samplers []*Sampler            `toml:"sampler"`
	Influxdb InfluxdbConfiguration `toml:"influxdb"`
	Monitor  MonitorConfig         `toml:"monitor"`
}

var config Config

func main() {

	// Read the configuration file
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		printConfig(err)
	}

	if _, err := os.Stat(config.Monitor.SocketPath); os.IsNotExist(err) {
		//we are daemon
		daemon()
	} else {
		//we are monitor
		monitor()
	}
}

func monitor() {
	monitor := NewMonitor(&config.Monitor)
	monitor.Start()
}

func daemon() {
	fmt.Printf("Initializing %d samplers...\n", len(config.Samplers))

	sampleChan := make(chan Sample)

	// Start a worker for each pipe
	for _, sampler := range config.Samplers {
		fmt.Printf("Initializing sampler, path: %s\n", sampler.Path)
		sampler.Init()
		go sampler.Collect(sampleChan)

	}

	fmt.Printf("All Initialized\n")

	influxdbReporter := New(&config.Influxdb)

	influxChan := make(chan Sample)

	go influxdbReporter.Listen(sampleChan)

	monitorReporter := NewMonitorReporter(&config.Monitor)

	monitorChan := make(chan Sample)

	go monitorReporter.Listen(sampleChan)

	for sample := range sampleChan {
		influxChan <- sample
		monitorChan <- sample
	}

}

func printConfig(err error) {
	if err != nil {
		fmt.Printf("An error parsing %s: %s\n", configPath, err.Error())
	}
	conf := `[[sampler]]
path = "/tmp/timing_log"
min = 1
max = 60000
sigint = 5

[influxdb]
url = "http://localhost:8086/"
username = "root"
password = "root"
database = "agento"
retentionPolicy = "default"
retries = 0`

	fmt.Printf("Write configuration file like this:\n---\n%s\n---\nsave in %s\n", conf, configPath)
	os.Exit(1)
}
