# Hardware construction

## Some pictures of the final installation

The two shelves in the hallway and the interrupting door

![Shelves in the hallway](images/shelves.jpg)

A view from the underside. You can see the LED stripe and one sensor
clearly. Note that the sensor is in fact not exactly positioned as
described in the config file (it should be located exactly at the
outer edge of the stripe. This bug is there for historic reasons and
will at one point in time changed by moving the sensor closer to the
edge).

![Underside of a shelf](images/shelf_underside.jpg)

Closeup of a sensor attached to a wooden square rod (covered in white
tape) that holds also the LED stripe

![Closeup of a sensor](images/ir-sensor.jpg)

The uninteresting looking box containing the Pi, the logic board and a
off the shelf power supply to drive everything, including the attached
LED stripes and the sensors.

![Closeup of the box (back)](images/goleds-box-back.jpg)

The uninteresting looking box from the other side. I am using a
centronics style connector to attach everything: Power for the LED
stripes & sensors, SPI lines for the stripes and 16 signal lines for
the sensors (The SPI goes to the ADCs which are in the case, not to
the outside sensors)

![Closeup of the box (front)](images/goleds-box-front.jpg)

## Raspberry Pi

I am using a Raspberry Pi 3b in my installation. This has plenty enough
horsepower to drive the project. In idle mode (no lights
burning/effect running, only the readout loop of the sensors is
active) the CPU usage is around 1.7%. While the multiblob effect is
running as configured in the default config file CPU usage goes up to
around 4.3%.

The OS used is standard Rasbian, and - thanks to the fact that GO
compiles to a static binary - there is not much else needed more than
that. You have to enable the SPI interface on the Pi though, there is
plenty of documentation on the net on other things you could do (like
overclocking, which may be overkill. One thing I did was making sure
that the CPU doesn't scale but instead keeps running at a steady
frequency as it seems to help with getting the SPI timings right, but
YMMV)

## The more interesting inside of the box

**BIG WARNING if you don't feel competent to work with mains voltage - DON'T DO IT**

**FOLLOWING WHAT I DID OR TRYING YOURSELF YOU ARE DOING AT YOUR OWN RISK**

**While I may (or may not) have a little clue about current and
voltage myself, I am in no means in the position to give anybody
reliable help on what or what not to do when it comes to dangerous
voltage and amp levels **

The next picture shows the opened box. The big thing closest to the
camera is the off the shelf power supply. It's 5V and enough amps to
drive the LED segments attached safely (and yes, also the Pi
itself and the sensors need their share of power).

Behind on the right is the Raspberry Pi 3b v1.2 used, and left next to
it is the custom multiplexer logic board.


![Closeup of the inside](images/goleds-box-open.jpg)


The next image show the connectors that are attached to the Pi (as far
as it can be seen there. The one and lonely connection at the left can
be ignored, this is the remains of an older experiment) 

![Closeup of the Raspberry Pi](images/goleds-box-raspi.jpg)

This connection routes the SPI GPIO pins and the GPIO pins used for
multiplexing the SPI port on the custom logic board (see below for
more info) 

### The custom logic board

![Closeup of the logic board](images/goleds-box-logic.jpg)



# **TODO**

- explain the custom logic board and its components
- electrical schematics


