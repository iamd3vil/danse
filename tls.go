package main

import (
	"crypto/tls"
	"encoding/binary"
	"io"
	"log"
	"net"

	"github.com/miekg/dns"
)

func (e *env) sendDOTQuery(b *dns.Msg) (*dns.Msg, error) {
	// Make a chan for reply
	replyChan := make(chan *dns.Msg)

	// Store reply channel in env
	e.tlsClients.Lock()
	e.tlsClients.clients[b.MsgHdr.Id] = replyChan
	e.tlsClients.Unlock()

	e.query <- b
	select {
	case r := <-replyChan:
		return r, nil
	}
}

func (e *env) sendQueries() {
	for m := range e.query {
		b, err := m.Pack()
		if err != nil {
			continue
		}
		length := len(b)
		size := make([]byte, 2)
		binary.BigEndian.PutUint16(size, uint16(length))
		_, err = e.tlsConn.Write(size)
		if err != nil {
			log.Println(err)
			e.connNotify <- 0
			break
		}
		_, err = e.tlsConn.Write(b)
		if err != nil {
			log.Println(err)
			e.connNotify <- 0
			break
		}
	}
}

func (e *env) readQueries() {
	for {
		size := make([]byte, 2)
		_, err := io.ReadFull(e.tlsConn, size)
		if err != nil {
			log.Println("Error while reading", err)
			e.connNotify <- 0
			break
		}

		resp := make([]byte, binary.BigEndian.Uint16(size))
		_, err = e.tlsConn.Read(resp)
		if err != nil {
			log.Println("Error while reading", err)
			e.connNotify <- 0
			break
		}

		m := &dns.Msg{}
		m.Unpack(resp)

		// Get reply chan
		e.tlsClients.Lock()
		rep := e.tlsClients.clients[m.MsgHdr.Id]
		rep <- m
		delete(e.tlsClients.clients, m.MsgHdr.Id)
		e.tlsClients.Unlock()
	}
}

func (e *env) makeTLSConnection(addr, name string) {
	log.Println("Starting TLS connection")
	for {
		connNotify := make(chan int, 2)
		e.connNotify = connNotify
		for {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				// If there is any error retry connecting
				log.Println("Error while connecting", err)
				continue
			}
			err = conn.(*net.TCPConn).SetKeepAlive(true)
			if err != nil {
				// If there is any error retry connecting
				log.Println("Error while connecting", err)
				continue
			}
			tConn := tls.Client(conn, &tls.Config{
				ServerName: name,
			})

			e.tlsConn = tConn
			break
		}
		log.Println("Connection formed")
		go e.sendQueries()
		go e.readQueries()

		// If conn notify gets a message, that means we have to try reconnecting
		<-e.connNotify
	}
}
