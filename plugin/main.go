package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/weaveworks/docker-plugin/plugin/driver"
	"github.com/weaveworks/docker-plugin/plugin/skel"
	. "github.com/weaveworks/weave/common"
)

var version = "(unreleased version)"

func main() {
	var (
		justVersion bool
		address     string
		nameserver  string
		debug       bool
	)

	flag.BoolVar(&justVersion, "version", false, "print version and exit")
	flag.BoolVar(&debug, "debug", false, "output debugging info to stderr")
	flag.StringVar(&address, "socket", "/run/docker/plugins/weave.sock", "socket on which to listen")
	flag.StringVar(&nameserver, "nameserver", "", "nameserver to provide to containers")

	flag.Parse()

	if justVersion {
		fmt.Printf("weave plugin %s\n", version)
		os.Exit(0)
	}

	if debug {
		SetLogLevel("debug")
	}

	var d skel.Driver
	d, err := driver.New(version, nameserver)
	if err != nil {
		Log.Fatalf("unable to create driver: %s", err)
	}

	var listener net.Listener

	listener, err = net.Listen("unix", address)
	if err != nil {
		Log.Fatal(err)
	}
	defer listener.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	endChan := make(chan error, 1)
	go func() {
		endChan <- skel.Listen(listener, d)
	}()

	select {
	case sig := <-sigChan:
		Log.Debugf("Caught signal %s; shutting down", sig)
	case err := <-endChan:
		if err != nil {
			Log.Errorf("Error from listener: ", err)
			listener.Close()
			os.Exit(1)
		}
	}
}
