# Go-LEDS

This project implements a multi-part LED stripe (ws2801) that is
controlled by multiple infrared sensors (Sharp GP2Y0A21YK0F or
similar) attached to analog-digital converters (MCP3008)
to turn on the LED stripes whenever someone is passing by. Sensors and
LED stripes are connected to a Raspberry Pi via SPI and a small board
of logic chips (AND and OR gates) to enable SPI multiplexing.

![Overview](images/overview.png)


## Example setup 

The specific layout used in my hallway is shown below:

| Stripe 1        | Door                 |        Stripe 2 |
|:----------------|:--------------------:|----------------:|
| LED 0-69        | "virtual LED" 70-110 |     LED 111-164 |
| Sensor S0 left  | (invisible segment)  |  Sensor S2 left |
| Sensor S1 right |                      | Sensor S3 right |

4 devices are connected via SPI (2 LED stripes, and 2 MCP3008 as
analog-digital converters (ADC) - these 2 ADCs allow for up to 16
sensors to be attached, although only 4 are used in my setup). The
reason 2 ADCs are used for 4 sensors is a left-over from the first
setup where 14 sensors were used spread out over the length of the two
LED stripes. This turned out to be problematic because of heavy
crosstalk between the sensors, so reducing them to be placed only at
the possible entry points for people passing by proofed to be enough.

Other setups may have a need for more sensors, so the hardware can
easily accomodate for 12 more input channels.

All hardware related stuff is held in the hardware package and is
configurable via the config file _config.yml_ - you can easily change
it to match your hardware (number and lenght of stripes, sensors,
placement of sensors, GPIO pins used for multiplexing etc.)

More drastic changes (other types of LED stripes, using other ADC
chips) may require changes in the code (_hardware/hardware.go_), or
even at the hardware level (attaching more than 4 devices e.g. more LED
stripes) by changing the multiplexing circuit.

## Mode of operation 

A couple of "producers" are supplied with the software (see the
directoy named accordingly) - these control different ways to illuminate
the stripes. The most important one is the sensorledproducer - each
sensor is linked to one instance of those. It reacts to a sensor
trigger by illuminating the stripe LED by LED starting from the
position of the sensor to both ends of the stripe. After a while, the
effect is reversed and the lighted area shrinks LED by LED until it
vanishes at the point where the sensor is located.

Other producers are explained in more detail below (**TODO**)



