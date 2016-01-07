package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/BurntSushi/toml"
)

const configPath = "/etc/lagpipe.conf"

type config struct {
	Samplers []*Sampler            `toml:"sampler"`
	Influxdb InfluxdbConfiguration `toml:"influxdb"`
	Monitor  MonitorConfig         `toml:"monitor"`
}

func main() {
	var config config

	// Read the configuration file
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		printConfig(err)
	}

	fmt.Printf("Initializing %d samplers...\n", len(config.Samplers))

	sampleChan := make(chan Sample)

	// Start a worker for each pipe
	for _, sampler := range config.Samplers {
		fmt.Printf("Initializing sampler, path: %s\n", sampler.Path)
		sampler.Init()
		go sampler.Collect(sampleChan)

	}
	fmt.Printf("All Initialized\n")

	var wg sync.WaitGroup

	influxdbReporter := New(&config.Influxdb)

	wg.Add(1)
	go influxdbReporter.Listen(sampleChan)

	monitorReporter := NewMonitorReporter(&config.Monitor)

	wg.Add(1)
	go monitorReporter.Listen(sampleChan)

	wg.Wait()
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
