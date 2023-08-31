# Go-LEDS

This Project implements a multi-part LED stripe (ws2801) that is
controlled by 4 infrared sensors attached to analog-digital converters
(MCP3008) to catch passing people with the net effect of illuminating
my hallway.  All sensors and LED stripes are connected to a Raspberry
Pi via SPI.

The specific layout used in my apartment is shown below:

| Stripe 1        | Door       | Stripe 2        |
|:----------------|:----------:|----------------:|
| 0-69            | (invisible | 70-124          |
| Sensor S0 left  | segment)   | Sensor S2 left  |
| Sensor S1 right |            | Sensor S3 right |

This can be easily changed in the config file _config.yml_. 4 devices are used via
SPI (2 led stripes, and 2 MCP3008 as ADC - these 2 ADCs allow for up
to 16 sensors to be attached, although only 4 are used in my setup)

A couple of "producers" are supplied (see the directoy named
accordingly) - these control various ways to illuminate the
stripes. The most important one is the sensorledproducer - each sensor
has one of these linked to itself. It react to a sensor trigger by
illuminating the stripe led by led starting from the position of the
sensor to both ends of the stripe. After a while, the effect is
reversed and the lighted area shrinks led by led until it vanishes at
the point where the sensor is located.

Other producers are explained in more detail below (**TODO**)

All hardware related stuff is held in the hardware package but mostly
configurable via the config file - you can easily change
it to match your hardware (number and lenght of stripes, sensors, placement of
sensors, GPIO pins used for multiplexing etc.)



               ╔══════════════════════════════════╗
               ║                                  ║
               ║     Raspberry Pi                 ║
               ║                                  ║
               ║                                  ║
               ╚══════════════════════════════════╝
                              ┃▲               ╏╏╏╏
                              ┃┃               ╏╏╏╏
                         SPI  ┃┃               ╏╏╏╏ GPIO
                              ┃┃               ╏╏╏╏
                              ┃┃               ╏╏╏╏
                              ▼┃               ▼▼▼▼
     ┏╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺┓
     ╏       Multiplexing via AND/OR gates       ╏
     ╏            driven by GPIO Pins            ╏
     ┗╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺╺┛
         ┃▲        ┃▲            ┃▲         ┃▲
         ┃┃        ┃┃            ┃┃         ┃┃
         ┃┃ SPI    ┃┃ SPI        ┃┃ SPI     ┃┃ SPI
         ┃┃        ┃┃            ┃┃         ┃┃
         ┃┃        ┃┃            ┃┃         ┃┃
         ▼┃        ▼┃            ▼┃         ▼┃
    ╭─────────╮╭─────────╮   ╭─────────╮╭─────────╮
    │         ││         │   │         ││         │
    │ LED 1   ││ LED 2   │...│ ADC 1   ││ ADC 2   │
    ╰─────────╯╰─────────╯   ╰─────────╯╰─────────╯

