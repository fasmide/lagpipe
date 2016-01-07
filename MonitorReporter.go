package main

import (
	"log"
	"net"
)

type MonitorConfig struct {
	SocketPath string `toml:"path"`
}

type MonitorReporter struct {
	clients []net.Conn
	config  *MonitorConfig
}

func NewMonitorReporter(c *MonitorConfig) MonitorReporter {
	log.Printf("Initializing MonitorReporter: %s", c.SocketPath)
	return MonitorReporter{config: c, clients: make([]net.Conn, 0, 10)}
}

func (m *MonitorReporter) Listen(ch chan Sample) {

	l, err := net.Listen("unix", m.config.SocketPath)

	if err != nil {
		log.Fatal("listen error:", err)
	}

	newConnections := make(chan net.Conn)
	//dont know why im doing this...
	go (func(ch chan net.Conn) {
		for {

			fd, err := l.Accept()
			if err != nil {
				log.Fatal("accept error:", err)
			}

			log.Printf("Someone connected to our monitor socket")

			ch <- fd
		}
	})(newConnections)

	for {
		select {

		case conn := <-newConnections:
			m.clients = append(m.clients, conn)
		case <-ch:
			for _, conn := range m.clients {
				conn.Write([]byte("der er en sample"))
			}
		}
	}
}
