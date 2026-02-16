# GoLEDS Project Context

## Project Overview
GoLEDS is a highly configurable, concurrent lighting system written in Go. It controls LED strips (like WS2801, APA102) based on infrared (IR) sensor inputs. The system is designed for Raspberry Pi but features a robust terminal-based simulation (TUI) for cross-platform development.

**Key Features:**
*   **Reactive Lighting:** Animations triggered by IR motion sensors.
*   **Ambient Modes:** Clocks, Nightlights (sunrise/sunset aware), Audio VU meters.
*   **Platform Abstraction:** Same code runs on hardware and in a terminal simulator.
*   **Hot Reloading:** Configuration changes apply instantly without restarting.
*   **Web Interface:** A built-in web dashboard for tweaking settings on the fly.

## Tech Stack
*   **Language:** Go (v1.24.0)
*   **Hardware Interface:** `github.com/stianeikeland/go-rpio/v4` (GPIO/SPI)
*   **TUI Library:** `github.com/rivo/tview` and `github.com/gdamore/tcell/v2`
*   **Audio:** `github.com/gordonklaus/portaudio` (CGO required)
*   **Config:** YAML via `gopkg.in/yaml.v3`
*   **Web:** Standard `net/http` with vanilla JS frontend.

## Architecture

### 1. Core Abstractions
The system revolves around two main interfaces:
*   **`platform.Platform`**: Abstracts the hardware layer.
    *   **`RaspberryPiPlatform`**: Drives SPI for LEDs and reads ADC (MCP3008) for sensors. Supports SPI multiplexing.
    *   **`TUIPlatform`**: Renders LEDs as colored text blocks and simulates sensors via keyboard input.
*   **`producer.LedProducer`**: Generates LED colors.
    *   Producers run concurrently.
    *   Outputs are combined (max value wins) to allow layering effects (e.g., a clock overlaying a nightlight).

### 2. State Management (`goleds.go`)
The main loop (`stateManager`) coordinates the "mood" of the system:
*   **Idle:** Permanent producers (Clock, Nightlight, Audio) are active.
*   **Sensor Triggered:** When a sensor fires, permanent producers stop, and the `SensorLedProducer` takes over (Run-Up -> Hold -> Run-Down).
*   **After Effects:** Once the sensor interaction ends, ambient effects (Cylon, MultiBlob) can play before returning to Idle.

### 3. Data Flow
`Platform` (Sensors) -> `App` (State Manager) -> `Producers` (Animation Logic) -> `AtomicEvent` -> `Platform` (Display Driver) -> `Hardware/Screen`.

## Directory Structure
*   `goleds.go`: Main entry point, signal handling, and state machine.
*   `platform/`: Hardware abstraction.
    *   `rpiplatform.go`: SPI/GPIO logic.
    *   `tuiplatform.go`: Simulation UI.
    *   `segment.go`: Logic for mapping virtual LED indices to physical segments.
*   `producer/`: Animation logic.
    *   `sensorledproducer.go`: The core reactive "pulse" animation.
    *   `multiblobproducer.go`: Physics-based colliding color blobs.
    *   `audioledproducer.go`: Audio-reactive VU meter.
*   `config/`: Configuration structs and validation.
    *   `webhandler.go`: API for the frontend.
*   `web/`: Static assets (`index.html`, `app.js`) for the configuration dashboard.

## Development Guide

### Building and Running
**Local Simulation (TUI):**

```bash
go build -o goleds
./goleds
```

*   **Controls:**
    *   `1-9`: Trigger sensors.
    *   `+`/`-`: Adjust simulated trigger threshold.
    *   `q`: Quit.

**Raspberry Pi (Cross-Compile):**
```bash
./buildpi.sh
# Transfer 'goleds_pi' and 'config.yml' to Pi
sudo chrt 99 ./goleds_pi -real
```

### Configuration (`config.yml`)
The `config.yml` is the brain of the operation. Key sections:
*   **`Hardware`**: SPI pins, LED type (WS2801/APA102), sensor mapping.
*   **`SensorLED`**: Timing for the main reactive animation (`RunUpDelay`, `HoldTime`, `RunDownDelay`).
*   **`NightLED`**: Lat/Long for sunset calculations.
*   **`AudioLED`**: Audio device name and frequency analysis settings.

**Runtime Updates:**
*   Edit `config.yml` manually: The app watches the file and reloads automatically.
*   Web Interface: Open `http://<pi-ip>:8080` to edit settings via a GUI.

### Adding a New Producer
1.  Create `producer/myproducer.go`.
2.  Implement the `LedProducer` interface.
3.  Embed `AbstractProducer` for free concurrent state handling.
4.  Add a configuration struct in `config/config.go`.
5.  Wire it up in `goleds.go`:
    *   Instantiate it in `initialise`.
    *   Add it to `permProd`, `afterProd`, or handle it in `stateManager` depending on when it should run.

## Validation & Dependencies

### Hardware Immutability
*   **`LedsTotal`**: This field represents the physical number of LEDs and is considered a read-only hardware attribute. The backend (`config/webhandler.go`) explicitly rejects runtime API requests that attempt to modify this value. This prevents validation errors where dependent fields (like `ClockLED` ranges) are validated against a stale hardware configuration.

### Producer Dependencies
*   **SensorLED Dependency**: The "After Producers" (currently `CylonLED` and `MultiBlobLED`) are logically dependent on the `SensorLED` producer. They only run *after* a sensor event completes.
    *   **Backend Rule**: `Config.Validate()` enforces that if `CylonLED` or `MultiBlobLED` are enabled, `SensorLED` must also be enabled.
    *   **UI Behavior**: Both the Web UI and Flutter App implement "Auto-disable" logic. Disabling `SensorLED` automatically disables and unchecks the dependent producers to ensure a valid configuration is sent to the backend.

## Common Tasks
*   **Calibrate Sensors:** Run `./goleds_pi -real -show-sensors` on the Pi to see raw ADC values and adjust `TriggerValue` in config.
*   **Change Colors:** Use the Web UI (`http://localhost:8080` if local) or edit `config.yml`.
*   **Debug:** Check `logging/` or stdout. The TUI has a scrolling log window.

## Important Notes
*   **Concurrency:** The system relies heavily on goroutines and channels. Use `util.AtomicEvent` for passing state between the high-speed display loop and slower logic loops.
*   **Performance:** On the Pi, `chrt 99` is critical for smooth LED timing, especially with WS2801 chips.
*   **Audio:** Requires `libportaudio2` installed on the system (`sudo apt install libportaudio2`).
