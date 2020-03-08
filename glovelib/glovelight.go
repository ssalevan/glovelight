package glovelib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/amimof/huego"
	"github.com/rs/zerolog/log"
	"gitlab.com/gomidi/midi/mid"
	"gitlab.com/gomidi/rtmididrv"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v2"
)

const GlovelightUser = "Glovelight User"

func logInPorts(ports []mid.In) {
	log.Info().Msgf("MIDI IN Ports:")
	for _, port := range ports {
		log.Info().Msgf("[%v] %s\n", port.Number(), port.String())
	}
}

type Glovelight struct {
	Controllers  []*Controller `yaml:"controllers"`
	BridgeIP     string `yaml:"bridge_ip"`
	User         string `yaml:"user"`
	location     string
	debugEnabled bool
	limiter     *rate.Limiter
	bridge       *huego.Bridge
	lights       []huego.Light
	stateChanges chan *lightStateChange
}

type lightStateChange struct {
	bulbID int
	state huego.State
}

func (g *Glovelight) setLightState(bulbID int, state huego.State) {
	g.stateChanges <- &lightStateChange{
		bulbID: bulbID,
		state: state,
	}
}

func (g *Glovelight) ConnectToMIDI(justLogPorts bool) error {
	drv, err := rtmididrv.New()
	if err != nil {
		return err
	}
	inPorts, err := drv.Ins()
	if err != nil {
		return err
	}
	if justLogPorts {
		logInPorts(inPorts)
		return nil
	}
	unknownMidiInputs := make([]string, 0)
	for _, controller := range g.Controllers {
		found := false
		for _, inPort := range inPorts {
			if inPort.String() == controller.MidiInput {
				controller.inPort = inPort
				found = true
			}
		}
		if !found && !stringInSlice(controller.MidiInput, unknownMidiInputs) {
			unknownMidiInputs = append(unknownMidiInputs, controller.MidiInput)
		}
	}
	if len(unknownMidiInputs) > 0 {
		return fmt.Errorf("Unknown MIDI inputs: %s", strings.Join(unknownMidiInputs, ", "))
	}
	return nil
}

func (g *Glovelight) ConnectToBridge() error {
	// Connects to the Hue bridge, discovering it then logging in if necessary.
	var err error
	if g.BridgeIP != "" && g.User != "" {
		log.Info().Msgf("Connecting to Hue bridge at: %s", g.BridgeIP)
		g.bridge = huego.New(g.BridgeIP, g.User)
	} else {
		log.Info().Msgf("Discovering Hue bridge...")
		g.bridge, err = huego.Discover()
		if err != nil {
			return err
		}
		log.Info().Msgf("Discovered Hue bridge: %+v", g.bridge)
		g.BridgeIP = g.bridge.Host

		log.Info().Msgf("Press the Link button on your Hue bridge then press Enter within 30 seconds.")
		fmt.Scanln()
		g.User, err = g.bridge.CreateUser(GlovelightUser)
		if err != nil {
			return err
		}
		// Persists newly-created user to disk for use in future control sessions.
		err = g.WriteToDisk()
		if err != nil {
			log.Error().Msgf("Could not write Glovelight to %s: %v", g.location, err)
		}
	}

	// Enumerates all Hue lights currently connected to the bridge.
	g.lights, err = g.bridge.GetLights()
	if err != nil {
		return err
	}
	log.Debug().Msg("Discovered lights:")
	for _, light := range g.lights {
		log.Debug().Msgf("%+v", light)
	}

	// Assigns lights to each Controller.
	unknownBulbIDs := make([]int, 0)
	for _, controller := range g.Controllers {
		for _, bulbID := range controller.BulbIds {
			found := false
			for _, light := range g.lights {
				if light.ID == bulbID {
					found = true
					controller.lights = append(controller.lights, light)
					break
				}
			}
			if !found && !intInSlice(bulbID, unknownBulbIDs) {
				unknownBulbIDs = append(unknownBulbIDs, bulbID)
			}
		}
	}
	if len(unknownBulbIDs) > 0 {
		unknownBulbIDsStr, _ := json.Marshal(unknownBulbIDs)
		return fmt.Errorf("Unknown bulb IDs: %s", unknownBulbIDsStr)
	}

	log.Info().Msgf("Successfully connected to bridge at: %s", g.bridge.Host)

	return nil
}

func (g *Glovelight) startStateChangeLimiter() {
	go func() {
		for {
			stateChange := <-g.stateChanges
			allow := g.limiter.Allow()
			if allow {
				if l := log.Debug(); l.Enabled() {
					l.Msgf("Received state change: %+v", stateChange)
				}
				resp, err := g.bridge.SetLightState(stateChange.bulbID, stateChange.state)
				if err != nil {
					log.Error().Msgf("Failed to set light state: %v, %v", resp, err)
				}
				if l := log.Debug(); l.Enabled() {
					l.Msgf("Hue bridge response: %v", resp)
				}
			}
		}
	}()
}

func (g *Glovelight) Start() error {
	g.startStateChangeLimiter()
	for _, controller := range g.Controllers {
		err := controller.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Glovelight) WriteToDisk() error {
	log.Info().Msgf("Writing Glovelight file to: %s", g.location)
	glovelightYaml, err := yaml.Marshal(g)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(g.location, glovelightYaml, 0644)
}

func ReadGlovelightFile(location string) (*Glovelight, error) {
	glovelightFile, err := os.Open(location)
	if err != nil {
		return nil, err
	}
	defer glovelightFile.Close()
	fileBytes, err := ioutil.ReadAll(glovelightFile)
	if err != nil {
		return nil, err
	}
	glovelight := &Glovelight{}
	err = yaml.Unmarshal(fileBytes, glovelight)
	if err != nil {
		return nil, err
	}
	glovelight.limiter = rate.NewLimiter(rate.Every(time.Second / 12), 12)
	glovelight.stateChanges = make(chan *lightStateChange, 12)
	glovelight.location = location
	for _, controller := range glovelight.Controllers {
		controller.limiter = rate.NewLimiter(rate.Every(time.Second / 12), 1)
		if controller.MidiInput == "" {
			controller.MidiInput = "Glover"
		}
		if controller.MidiChannel == 0 {
			controller.MidiChannel = 1
		}
		controller.glovelight = glovelight
	}
	return glovelight, nil
}
