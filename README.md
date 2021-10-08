# Go-LEDS

This Project implements a two-part LED stripe that is controlled by 4
infrared sensors to catch passing people with the net effect to
illuminate my hallway.  All sensors or LED stripes are connected to a
Raspberry Pi via SPI. The layout is shown below:

| Stripe 1        | Door | Stripe 2        |
|:---------------:|:----:|:---------------:|
| 0-69            |      | 70-124          |
| Sensor S0 left  |      | Sensor S2 left  |
| Sensor S1 right |      | Sensor S3 right |

**All hardware related stuff is held in the hardware package - if you
know what you do you can easily change this to match your hardware
(number of stripes, sensors, placement of sensors, GPIO pins etc.)**
