package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/codahale/hdrhistogram"
)

type Sampler struct {
	Path             string `toml:"path"`
	Min              int64  `toml:"min"`
	Max              int64  `toml:"max"`
	Sigint           int    `toml:"sigint"`
	Name             string `toml:"name"`
	histogram        *hdrhistogram.Histogram
	bucketQuantiles  []float64
	outOfRangeErrors int64
}

type Sample struct {
	Quantiles            []Quantile
	Max, Min, TotalCount int64
	StdDev, Mean         float64
	OutOfRangeCount      int64
	Time                 time.Time
	Name                 string
}

type NginxSample struct {
	/*'$status $bytes_sent '
	  '$request_time $upstream_response_time';
	*/
	Status, BytesSent                 int64
	RequestTime, UpstreamResponseTime float64
}

type Quantile struct {
	Quantile float64
	ValueAt  int64
}

func (s *Sampler) Init() {
	s.outOfRangeErrors = 0
	s.histogram = hdrhistogram.New(s.Min, s.Max, s.Sigint)

	// we need to know the bucket Quantiles, for this we just add every possible value to the histogram..
	for i := s.Min; i < s.Max; i++ {
		s.histogram.RecordValue(i)
	}

	//We then extract hdrhistograms own bucket range
	distribution := s.histogram.CumulativeDistribution()
	s.bucketQuantiles = make([]float64, 0, len(distribution))

	//and save it for later use
	for _, d := range distribution {
		s.bucketQuantiles = append(s.bucketQuantiles, d.Quantile)
	}

	s.histogram.Reset()

	// Pipe stuff
	// Check if pipe already exists
	pipeExists := false
	fileInfo, err := os.Stat(s.Path)

	if err == nil {
		if (fileInfo.Mode() & os.ModeNamedPipe) > 0 {
			pipeExists = true
		} else {
			fmt.Printf("%d != %d\n", os.ModeNamedPipe, fileInfo.Mode())
			panic(s.Path + " exists, but it's not a named pipe (FIFO)")
		}
	}

	// Try to create pipe if needed
	if !pipeExists {
		err := syscall.Mkfifo(s.Path, 0666)
		if err != nil {
			panic(err.Error())
		}
	}

}

func (s *Sampler) sample(ch chan NginxSample) {

	// Open pipe for reading
	fd, err := os.Open(s.Path)
	if err != nil {
		panic(err.Error())
	}
	defer fd.Close()
	reader := bufio.NewReader(fd)

	measurement := NginxSample{}

	// Loop forever
	for {
		message, err := reader.ReadString(0xa)
		if err != nil && err != io.EOF {
			panic("Reading from pipe failed: " + err.Error())
		}

		if message != "" {

			measurement.set(message)

			ch <- measurement
		}
	}

}

func (s *Sampler) makeSample() Sample {

	//we should make a sample out of the current histogram...
	quantiles := []Quantile{}

	for _, quantile := range s.bucketQuantiles {
		quantiles = append(quantiles, Quantile{Quantile: quantile, ValueAt: s.histogram.ValueAtQuantile(quantile)})
	}

	return Sample{OutOfRangeCount: s.outOfRangeErrors,
		Quantiles:  quantiles,
		Max:        s.histogram.Max(),
		Min:        s.histogram.Min(),
		Mean:       s.histogram.Mean(),
		TotalCount: s.histogram.TotalCount(),
		StdDev:     s.histogram.StdDev(),
		Time:       time.Now(),
		Name:       s.Name,
	}

}

func (s *Sampler) Collect(ch chan Sample) {

	sampleChan := make(chan NginxSample)

	go s.sample(sampleChan)

	ticker := time.Tick(time.Minute)

	for {
		select {
		case nSample := <-sampleChan:

			e := s.histogram.RecordValue(int64(nSample.UpstreamResponseTime * 1000.0))

			if e != nil {
				s.outOfRangeErrors++
			}

		case <-ticker:
			//we should submit a sample
			ch <- s.makeSample()
			s.histogram.Reset()
			s.outOfRangeErrors = 0
		}
	}

}

func (m *NginxSample) set(s string) *NginxSample {
	i := strings.Fields(s)

	if len(i) > 3 {

		m.Status, _ = strconv.ParseInt(i[0], 10, 64)
		m.BytesSent, _ = strconv.ParseInt(i[1], 10, 64)

		m.RequestTime, _ = strconv.ParseFloat(i[2], 64)
		m.UpstreamResponseTime, _ = strconv.ParseFloat(i[3], 64)
	}

	return m
}
