package main

import (
	"fmt"
	"log"
	"net"
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

	//we should also try ot connect to the socket....

	if conn, err := net.Dial("unix", config.Monitor.SocketPath); err == nil {
		//we are monitor
		monitor(&conn)

	} else {
		//we are daemon
		daemon()
	}
}

func monitor(m *net.Conn) {
	monitor := NewMonitor(&config.Monitor)
	monitor.Start(m)
}

func daemon() {
	log.Printf("Making sure %s does not exist by removing it", config.Monitor.SocketPath)

	err := os.Remove(config.Monitor.SocketPath)

	if err != nil {
		log.Fatal("Could not remove %s: %s", config.Monitor.SocketPath, err.Error())
	}

	log.Printf("Initializing %d samplers...\n", len(config.Samplers))

	sampleChan := make(chan Sample)

	// Start a worker for each pipe
	for _, sampler := range config.Samplers {
		log.Printf("Starting sampler %s, path: %s\n", sampler.Name, sampler.Path)
		sampler.Init()
		go sampler.Collect(sampleChan)

	}

	log.Printf("All Initialized\n")

	influxdbReporter := NewInfluxdbReporter(&config.Influxdb)
	monitorReporter := NewMonitorReporter(&config.Monitor)

	for sample := range sampleChan {
		go influxdbReporter.report(sample)
		go monitorReporter.report(sample)
	}

}

func printConfig(err error) {
	if err != nil {
		fmt.Printf("An error parsing %s: %s\n", configPath, err.Error())
	}
	conf := `[[sampler]]
path = "/tmp/timing_log"
name = "some_name"
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
