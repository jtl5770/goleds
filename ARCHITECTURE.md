## Project Overview
GoLEDS is a highly configurable, concurrent lighting system written in Go. It controls LED strips (like WS2801, APA102) based on infrared (IR) sensor inputs. The system is designed for Raspberry Pi but features a robust terminal-based simulation (TUI) for cross-platform development.

**Key Features:**
*   **Reactive Lighting:** Animations triggered by IR motion sensors.
*   **Ambient Modes:** Clocks, Nightlights (sunrise/sunset aware), Audio VU meters.
*   **Audio VU Visualizer:** Pure Go LMS (Logitech / Lyrion Media Server) Squeezebox SlimProto visualizer with peak hold, linear falloff, and network auto-sync.
*   **Platform Abstraction:** Same code runs on hardware and in a terminal simulator.
*   **Hot Reloading:** Configuration changes apply instantly without restarting.
*   **Mobile Interface:** A Flutter-based dashboard/mobile app for tweaking settings on the fly.

## Tech Stack
*   **Language:** Go (v1.24.0)
*   **Hardware Interface:** `github.com/stianeikeland/go-rpio/v4` (GPIO/SPI)
*   **TUI Library:** `github.com/rivo/tview` and `github.com/gdamore/tcell/v2`
*   **Audio Engine:** Pure Go Squeezebox SlimProto & FLAC decoder (`github.com/mewkiz/flac`) with LMS JSON-RPC control (Zero CGO / no external audio libraries required)
*   **Config:** YAML via `gopkg.in/yaml.v3`
*   **Web:** Standard `net/http` with Flutter Web and REST API.

## Architecture

### 1. Core Abstractions
The system revolves around two main interfaces:
*   **`platform.Platform`**: Abstracts the hardware layer.
    *   **`RaspberryPiPlatform`**: Drives SPI for LEDs and reads ADC (MCP3008) for sensors. Supports SPI multiplexing.
    *   **`TUIPlatform`**: Renders LEDs as colored text blocks and simulates sensors via keyboard input.
*   **`producer.LedProducer`**: Generates LED colors.
    *   Producers run concurrently.
    *   Outputs are combined (max value wins) to allow layering effects (e.g., a clock overlaying a nightlight).

### 2. Audio Subsystem (`audio/`)
GoLEDS includes a high-performance, real-time stereo audio visualizer integrated with **LMS (Lyrion / Logitech Media Server)**:

*   **SlimProto Client (`audio/squeezebox/slimproto`)**:
    *   Connects directly to LMS on port 3483 over TCP as a virtual Squeezebox player.
    *   Receives SlimProto streaming commands (`strm s/p/u/q/f/a/t`) and HTTP audio streams.
    *   Decodes FLAC and raw PCM audio streams in pure Go without external C libraries.
    *   Feeds a lock-free circular ring buffer and tracks playback in real-time pace.
    *   Computes calibrated RMS dB levels per channel via `audio/dsp`.
    *   Includes automatic TCP socket reconnection and HELO handshakes.
*   **AutoSync Manager (`audio/squeezebox/control`)**:
    *   Queries LMS JSON-RPC (port 9000) to discover active players on the local network.
    *   Automatically joins the sync group of whichever player is playing music.
    *   Configures LMS player preferences (`maintainSync=0`, `minSyncAdjust=5000`) so GoLEDs acts as a purely passive visualizer and never introduces latency or adjustments to the master audio players.
    *   Automatically detects when a player pauses or stops, switching smoothly to new active players.
*   **Audio Provider (`audio/squeezebox/provider.go`)**:
    *   Lifecycle orchestrator managing the SlimProto client, AutoSyncManager, and dynamic runtime configuration updates.

### 3. State Management (`goleds.go`)
The main loop (`stateManager`) coordinates the "mood" of the system:
*   **Idle:** Permanent producers (Clock, Nightlight, Audio) are active.
*   **Sensor Triggered:** When a sensor fires, permanent producers stop, and the `SensorLedProducer` takes over (Run-Up -> Hold -> Run-Down).
*   **After Effects:** Once the sensor interaction ends, ambient effects (Cylon, MultiBlob) can play before returning to Idle.

### 4. Data Flow
`Platform` (Sensors) / `AudioProvider` (LMS Stream) -> `App` (State Manager) -> `Producers` (Animation Logic) -> `AtomicEvent` -> `Platform` (Display Driver) -> `Hardware/Screen`.

## Directory Structure
*   `goleds.go`: Main entry point, signal handling, and state machine.
*   `audio/`: LMS Squeezebox integration and DSP audio analysis.
    *   `dsp/`: Decibel calculation and RMS volume analysis.
    *   `squeezebox/`: Squeezebox audio provider orchestration.
    *   `squeezebox/control/`: LMS JSON-RPC client and AutoSyncManager.
    *   `squeezebox/slimproto/`: Squeezebox protocol client, HTTP FLAC streaming & ring buffer.
*   `platform/`: Hardware abstraction.
    *   `rpiplatform.go`: SPI/GPIO logic.
    *   `tuiplatform.go`: Simulation UI.
    *   `segment.go`: Logic for mapping virtual LED indices to physical segments.
*   `producer/`: Animation logic.
    *   `sensorledproducer.go`: The core reactive "pulse" animation.
    *   `multiblobproducer.go`: Physics-based colliding color blobs.
    *   `audioledproducer.go`: Stereo VU meter with dynamic peak hold and zone falloff.
*   `config/`: Configuration structs and validation.
    *   `webhandler.go`: REST API for the frontend and mobile app.
*   `web/`: Static assets for the Web UI.
*   `mobile/`: Flutter-based GoLEDS Commander application (Android, Linux, Web).

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
*   **`AudioLED`**: Audio visualizer configuration:
    *   `Enabled`: Enable/disable audio visualizer.
    *   `Server`: LMS server hostname/IP (e.g. `lms.local` or `192.168.1.50`).
    *   `SlimprotoPort`: SlimProto TCP port (default `3483`).
    *   `CliPort`: LMS JSON-RPC port (default `9000`).
    *   `AutoSync`: Automatically discover and synchronize with active LMS players.
    *   `PlayerName` & `Mac`: Custom visualizer client name and virtual MAC address.
    *   `MinDB` / `MaxDB`: Decibel range mapping (e.g. `-45.0` dB to `0.0` dB).
    *   `StartLedLeft` / `EndLedLeft` & `StartLedRight` / `EndLedRight`: LED ranges for left and right channels.
    *   `LedGreen`, `LedYellow`, `LedRed`: Multi-zone gradient bar colors.
    *   `PeakHoldTime`: Duration in milliseconds before peak indicators begin falling (e.g. `60` ms).
    *   `PeakDecayRate`: Linear falloff speed in LEDs per second (e.g. `20.0` LEDs/sec).
    *   `Dynamic Zone Peak Colors`: Peak indicators automatically calculate brightened colors (1.8×) corresponding to the zone (green, yellow, red) where the peak was set, and retain that color as they decay.

**Runtime Updates:**
*   Edit `config.yml` manually: The app watches the file and reloads automatically.
*   Web Interface & App: Open `http://<pi-ip>:8080` or use GoLEDS Commander to edit settings via GUI.

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
    *   **UI Behavior**: The Flutter App implements "Auto-disable" logic. Disabling `SensorLED` automatically disables and unchecks the dependent producers to ensure a valid configuration is sent to the backend.

## Common Tasks
*   **Calibrate Sensors:** Run `./goleds_pi -real -show-sensors` on the Pi to see raw ADC values. The system has an auto calibration run once on startup. This can also be triggered via the Web UI / mobile app.
*   **Change Colors:** Use the Web UI (`http://localhost:8080` if local), GoLEDS Commander app, or edit `config.yml`.
*   **Debug:** Check `logging/` or stdout. The TUI has a scrolling log window.

## Important Notes
*   **Concurrency:** The system relies heavily on goroutines and channels. Use `util.AtomicEvent` for passing state between the high-speed display loop and slower logic loops.
*   **Performance:** On the Pi, `chrt 99` is critical for smooth LED timing, especially with WS2801 chips.
*   **Audio:** Pure Go implementation without external C/CGO dependencies. Fully cross-platform.
