package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/kubearmor/KubeArmor/protobuf"
	"google.golang.org/grpc"
)

func GetOSSigChannel() chan os.Signal {
	c := make(chan os.Signal, 1)

	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		os.Interrupt)

	return c
}

func StartKubeArmorRelay(StopChan chan struct{}) {
	conn, err := grpc.Dial("localhost:32767", grpc.WithInsecure())
	if err != nil {
		fmt.Print(err.Error())
		return
	}

	client := pb.NewLogServiceClient(conn)

	req := pb.RequestMessage{}

	//Stream Logs
	go func() {
		defer conn.Close()
		if stream, err := client.WatchLogs(context.Background(), &req); err == nil {
			for {
				select {
				case <-StopChan:
					return

				default:
					res, err := stream.Recv()
					if err != nil {
						fmt.Print("system log stream stopped: " + err.Error())
					}

					fmt.Printf("Log %v\n", res)
				}
			}
		} else {
			fmt.Print("unable to stream systems logs: " + err.Error())
		}
	}()

	//Stream Alerts
	go func() {
		defer conn.Close()
		if stream, err := client.WatchAlerts(context.Background(), &req); err == nil {
			for {
				select {
				case <-StopChan:
					return

				default:
					res, err := stream.Recv()
					if err != nil {
						fmt.Print("system alerts stream stopped: " + err.Error())
					}

					fmt.Printf("Alert %v\n", res)
				}
			}
		} else {
			fmt.Print("unable to stream systems alerts: " + err.Error())
		}
	}()
}

func main() {
	SystemStopChan := make(chan struct{})
	go StartKubeArmorRelay(SystemStopChan)
	sigChan := GetOSSigChannel()
	<-sigChan
	close(SystemStopChan)
}
