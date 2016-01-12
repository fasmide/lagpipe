package main

import (
	"encoding/json"
	"log"
	"net"
	"sync"
)

type MonitorConfig struct {
	SocketPath string `toml:"path"`
}

type MonitorReporter struct {
	clients    []net.Conn
	config     *MonitorConfig
	lock       sync.Mutex
	lastSample *Sample
}

func NewMonitorReporter(c *MonitorConfig) *MonitorReporter {
	log.Printf("Initializing MonitorReporter: %s", c.SocketPath)
	m := MonitorReporter{config: c, clients: make([]net.Conn, 0, 10)}

	l, err := net.Listen("unix", m.config.SocketPath)

	if err != nil {
		log.Fatal("listen error:", err)
	}

	go func() {
		for {
			fd, err := l.Accept()

			if err != nil {
				log.Fatal("accept error:", err)
			}

			log.Printf("Someone connected to our monitor socket")

			m.lock.Lock()
			m.clients = append(m.clients, fd)
			m.lock.Unlock()

			m.send(m.lastSample, fd)
		}
	}()

	return &m
}

func (m *MonitorReporter) send(sample *Sample, fd net.Conn) {

	payload, _ := json.Marshal(sample)

	_, err := fd.Write(payload)

	if err != nil {
		m.remove(fd)
	}

}

func (m *MonitorReporter) remove(fd net.Conn) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i, conn := range m.clients {
		if conn == fd {
			m.clients = append(m.clients[:i], m.clients[i+1:]...)
			break
		}
	}
}

func (m *MonitorReporter) report(sample Sample) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.lastSample = &sample

	for _, conn := range m.clients {
		go m.send(&sample, conn)
	}

}
