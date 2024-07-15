package main

import (
	"github.com/ArthurHlt/go-eureka-client/eureka"
	"google.golang.org/grpc"
	"log"
	"net"
	ssov1 "pgk-schedule-script/gen/go"
	"pgk-schedule-script/scripts"
	"strconv"
)

const (
	port    = 50053
	appName = "pgk-schedule-script-service"
)

func main() {

	err := RegisterEurekaClient()
	if err != nil {
		log.Fatal(err)
		return
	}

	s := grpc.NewServer()
	src := &scripts.ScheduleScriptServiceServer{}
	ssov1.RegisterScheduleScriptServiceServer(s, src)

	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatal(err)
	}

	if err := s.Serve(l); err != nil {
		log.Fatal(err)
	}
}

func RegisterEurekaClient() error {
	client := eureka.NewClient([]string{
		"http://127.0.0.1:8761/eureka",
	})

	instance := eureka.NewInstanceInfo("localhost", appName, "127.0.0.1", port, 30, false)
	instance.Metadata = &eureka.MetaData{
		Map: make(map[string]string),
	}
	instance.Metadata.Map["version"] = "1.0.0"

	err := client.RegisterInstance(appName, instance)
	if err != nil {
		return err
	}
	return nil
}
