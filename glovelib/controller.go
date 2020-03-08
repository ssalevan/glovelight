package glovelib

import (
	"encoding/json"

	"github.com/amimof/huego"
	"github.com/rs/zerolog/log"
	"gitlab.com/gomidi/midi/mid"
	"golang.org/x/time/rate"
)

const MaxCCValue = 127

type Controller struct {
	BulbIds     []int  `yaml:"bulb_ids"`
	MidiInput   string `yaml:"midi_input"`
	MidiChannel uint8  `yaml:"midi_channel"`
	XCC         uint8  `yaml:"x_cc"`
	YCC         uint8  `yaml:"y_cc"`
	haveXVal    bool
	xVal        float32
	haveYVal    bool
	yVal        float32
	inPort      mid.In
	limiter     *rate.Limiter
	lights      []huego.Light
	glovelight  *Glovelight
}

func (c *Controller) XYVal() []float32 {
	return []float32{c.xVal, c.yVal}
}

func (c *Controller) setLightXY(lightID int) {
	c.glovelight.setLightState(lightID, huego.State{On: true, Xy: c.XYVal()})
}

func (c *Controller) HandleCC(p *mid.Position, channel, controller, value uint8) {
	if l := log.Debug(); l.Enabled() {
		l.Msgf("Received CC channel=%d controller=%d value=%d", channel, controller, value)
	}
	if channel == c.MidiChannel - 1 && controller == c.XCC {
		c.haveXVal = true
		c.xVal = float32(value) / MaxCCValue
		if l := log.Debug(); l.Enabled() {
			l.Msgf("Received X=%f for Controller: %+v", c.xVal, c)
		}
	} else if channel == c.MidiChannel - 1 && controller == c.YCC {
		c.haveYVal = true
		c.yVal = float32(value) / MaxCCValue
		if l := log.Debug(); l.Enabled() {
			l.Msgf("Received Y=%f for Controller: %+v", c.yVal, c)
		}
	}
	if c.limiter.Allow() && c.haveXVal && c.haveYVal {
		for _, light := range c.lights {
			if l := log.Debug(); l.Enabled() {
				bulbIdsStr, _ := json.Marshal(c.BulbIds)
				l.Msgf("Sending X=%f, Y=%f to bulbs: %s", c.xVal, c.yVal, bulbIdsStr)
			}
			c.setLightXY(light.ID)
		}
	}
}

func (c *Controller) Start() error {
	log.Debug().Msgf("Controller starting: %+v", c)
	// Connects MIDI listener to selected MIDI input port.
	err := c.inPort.Open()
	if err != nil {
		return err
	}
	rd := mid.NewReader(mid.NoLogger())
	mid.ConnectIn(c.inPort, rd)
	rd.Msg.Channel.ControlChange.Each = c.HandleCC
	// Turns on all the Hue lights attached to this Controller.
	for _, light := range c.lights {
		err = light.On()
		if err != nil {
			return err
		}
	}
	return nil
}
