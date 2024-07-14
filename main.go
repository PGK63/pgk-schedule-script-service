package main

import (
	"google.golang.org/grpc"
	"log"
	"net"
	ssov1 "pgk-schedule-script/gen/go"
	"pgk-schedule-script/scripts"
)

const (
	port = ":50053"
)

func main() {
	s := grpc.NewServer()
	src := &scripts.ScheduleScriptServiceServer{}
	ssov1.RegisterScheduleScriptServiceServer(s, src)

	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal(err)
	}

	if err := s.Serve(l); err != nil {
		log.Fatal(err)
	}
}
