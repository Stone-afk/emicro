package emicro

import "net"

type Server struct {
}

func (s *Server) Start(address string) error {
	return nil
}

func (s *Server) handleConn(conn net.Conn) error {
	return nil
}

type reflectionStub struct {
}