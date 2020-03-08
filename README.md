Glovelight - Hue Connector for MI.MU Gloves
===========================================

Have you ever purchased a set of [MI.MU Gloves](https://mimugloves.com) and wondered if
you could connect them up to your [Hue lights](https://www2.meethue.com)? Well, we over
at [Vox Cadre](https://voxcadre.com) did as part of our MI.MU Gloves-oriented
[100 Day Project](https://instagram.com/voxcadre). We hacked this tool out one evening
and figured that it might be useful for anyone who might enjoy this combination.

Limitations
-----------

The Hue bridge can become overwhelmed if too many light changes are sent to it
simultaneously, leading to it rejecting said changes. Glovelight was written to
limit latency as much as possible; therefore, it will rate limit light state changes
to 12 per second, immediately rejecting any changes that exceed this rate.

In general, we've seen about 100 milliseconds of latency between a Glove-effected
state change and a corresponding Hue color change.

Installation
------------

Glovelight was written in Golang and, after [installing Golang](https://golang.org/dl/),
you can install Glovelight with the following command:

`$ go get github.com/ssalevan/glovelight`

Configuration
-------------

Glovelight uses the X/Y-based control mechanism for selecting colors, which maps the color
of a bulb to an X/Y coordinate within the
[CIE 1931 color space](https://en.wikipedia.org/wiki/CIE_1931_color_space).

To map inputs from your MI.MU Gloves to X/Y color coordinates which select light colors,
Glovelight needs to know a few bits of information:

- The name of the Glover MIDI input (defaults to `Glover`)
- The MIDI channel where MI.MU Glove input will be received (defaults to `1`)
- The numeric Hue Bulb IDs to control (can be found under Settings->Light Setup in the Hue app)
- The MIDI CCs where X and Y input will be received

You'll want to place all of this information into something called a **Glovelight file**,
which is simply a [YAML](https://en.wikipedia.org/wiki/YAML) file. An example which controls
Hue bulbs 7 and 8, using MIDI CC 67 for X coordinate data and MIDI CC 68 for Y coordinate data
looks something like this:

```yaml
controllers:
  - bulb_ids:
      - 7
      - 8
    midi_channel: 1
    x_cc: 67
    y_cc: 68
```

Be sure to save this file somewhere easy to remember as you'll need it for the next step.

Running Glovelight
------------------

Once you've rigged up a Glovelight file, it's as simple as executing the following command:

`$ glovelight <path to your Glovelight file>`

If you haven't specified a `bridge_ip` and `user` in your Glovelight configuration, it will
attempt to discover any Hue bridge on your local network. If it finds said bridge, you'll
want to press the link button then hit Enter within 30 seconds to pair Glovelight with it.

At this point, you should be ready to go! Make sure that you've configured Glover to send
CCs corresponding to the `x_cc` and `y_cc` values you've specified in the configuration.

If you'd like to see further debugging logging output (helpful for determining if MIDI
data is coming through), you can pass the `-debug` flag to Glovelight. Furthermore, if you'd
like to see all the MIDI device names known to Glovelight you can pass the `-justLogPorts`
flag, which will list said device names. This mode can be useful if you'd like to take
advantage of the `midi_input` configuration option.

Configuration Reference
-----------------------

Further configuration options, beyond those above, are available. Here's an example of a
fully-specced out Glovelight file:

```yaml
controllers: # List of MIDI input pairings to Hue bulb IDs.
  - bulb_ids: # numerical Hue bulb identifiers
      - 1
      - 2
    midi_input: Glover  # name of MIDI device (default: Glover)
    midi_channel: 1     # channel on MIDI device (default: 1)
    x_cc: 67            # MIDI CC number for X coordinate
    y_cc: 68            # MIDI CC number for Y coordinate
  - bulb_ids:
      - 3
      - 4
    midi_input: Phil Collins
    midi_channel: 1
    x_cc: 69
    y_cc: 70
bridge_ip: 192.168.2.112 # IP address of Hue bridge (if not using auto-discover)
user: abcdefghijklmnopqrstuvwxyz # Username for Hue bridge (if not using auto-discover)
```

Comments
--------

Feel free to drop a GitHub issue our way or shoot us an e-mail at info@voxcadre.com.

Happy Glovelighting!
