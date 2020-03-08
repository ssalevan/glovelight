package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/ssalevan/glovelight/glovelib"
)

var debug = flag.Bool("debug", false, "enables debugging logs")
var justLogPorts = flag.Bool("justLogPorts", false,
	"if enabled, just logs MIDI input ports")

func main() {
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	glovelightFileArg := 1
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		glovelightFileArg++
	}
	if *justLogPorts {
		glovelightFileArg++
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if len(os.Args) < glovelightFileArg + 1 {
		fmt.Println("Usage: glovelight [-debug] <path to Glovelight file>")
		os.Exit(-1)
	}
	glovelightFile := os.Args[glovelightFileArg]
	glovelight, err := glovelib.ReadGlovelightFile(glovelightFile)
	if err != nil {
		fmt.Println("Could not read Glovelight file:", err)
		os.Exit(-2)
	}
	err = glovelight.ConnectToMIDI(*justLogPorts)
	if err != nil {
		fmt.Println("Could not connect to MIDI input:", err)
		os.Exit(-3)
	}
	if *justLogPorts {
		os.Exit(0)
	}

	err = glovelight.ConnectToBridge()
	if err != nil {
		fmt.Println("Could not connect to bridge:", err)
		os.Exit(-4)
	}

	err = glovelight.Start()
	if err != nil {
		fmt.Println("Unable to begin Glovelighting:", err)
		os.Exit(-5)
	}

	log.Info().Msg("Glovelight started; awaiting SIGINT...")

	intSignal := make(chan os.Signal, 1)
	signal.Notify(intSignal, os.Interrupt)

	<-intSignal
	log.Info().Msg("Glovelight received SIGINT, shutting down...")
}
