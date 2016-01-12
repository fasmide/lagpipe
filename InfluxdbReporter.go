package main

import (
	"fmt"
	"os"

	"github.com/influxdata/influxdb/client/v2"
)

type InfluxdbConfiguration struct {
	Url             string `toml:"url"`
	Username        string `toml:"username"`
	Password        string `toml:"password"`
	Database        string `toml:"database"`
	RetentionPolicy string `toml:"retentionPolicy"`
	Retries         int    `toml:"retries"`
}

type InfluxdbReporter struct {
	client   client.Client
	config   *InfluxdbConfiguration
	hostname string
}

func NewInfluxdbReporter(c *InfluxdbConfiguration) InfluxdbReporter {

	lort := client.HTTPConfig{
		Addr:     c.Url,
		Username: c.Username,
		Password: c.Password}

	cli, err := client.NewHTTPClient(lort)

	if err != nil {
		fmt.Printf("Chould not create influx http client: %s\n", err.Error())
		panic("No influxdb no game...")
	}

	hostname, err := os.Hostname()

	if err != nil {
		fmt.Printf("Could not determine hostname: %s\n", err.Error())
		panic("Too bad")
	}

	return InfluxdbReporter{client: cli, config: c, hostname: hostname}
}

func (i *InfluxdbReporter) Listen(sampleChan chan Sample) {

	for {
		sample := <-sampleChan
		fmt.Printf("Ny sample: %+v\n\n", sample)
		go i.report(sample)
	}
}

func (i *InfluxdbReporter) report(sample Sample) {
	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: i.config.Database,
	})

	if err != nil {
		fmt.Printf("Cannot create influx batch: %s\n", err.Error())
		panic("No batch no fun")
	}

	/*
	   {Quantiles:[
	   {Quantile:0 ValueAt:0} {Quantile:50 ValueAt:14}
	   {Quantile:75 ValueAt:15} {Quantile:87.5 ValueAt:18}
	   {Quantile:93.75 ValueAt:21} {Quantile:96.875 ValueAt:23}
	   {Quantile:98.4375 ValueAt:25} {Quantile:99.21875 ValueAt:27}
	   {Quantile:99.609375 ValueAt:28} {Quantile:99.8046875 ValueAt:29}
	   {Quantile:99.90234375 ValueAt:30} {Quantile:99.951171875 ValueAt:32}
	   {Quantile:99.9755859375 ValueAt:36} {Quantile:99.98779296875 ValueAt:42}
	   {Quantile:99.993896484375 ValueAt:45} {Quantile:99.9969482421875 ValueAt:45}
	   {Quantile:99.99847412109375 ValueAt:45} {Quantile:100 ValueAt:45}]
	   Max:45 Min:4 TotalCount:22815
	   StdDev:3.417546635937428 Mean:14.424194608809993 OutOfRangeCount:0}
	*/

	// Create a point and add to batch
	tags := map[string]string{"host": i.hostname, "lagpipe_name": sample.Name}
	fields := map[string]interface{}{
		"max":             sample.Max,
		"min":             sample.Min,
		"totalcount":      sample.TotalCount,
		"stddev":          sample.StdDev,
		"mean":            sample.Mean,
		"outofrangecount": sample.OutOfRangeCount,
	}

	for _, quantile := range sample.Quantiles {
		fields[fmt.Sprintf("p%f", quantile.Quantile)] = quantile.ValueAt
	}

	pt, _ := client.NewPoint("lagpipe", tags, fields, sample.Time)
	bp.AddPoint(pt)

	// Write the batch
	i.client.Write(bp)
}
