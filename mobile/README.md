# GoLEDS Commander

**GoLEDS Commander** is the remote management interface for the [GoLEDS](https://github.com/jtl5770/goleds)
reactive lighting system. Built with [Flutter](https://flutter.dev), it provides a unified,
beautiful UI to control your LED strips from Android, Linux, or the
Web.

![GoLEDS Commander](images/sensors.png)

## Features

*   **Real-time Control:** Toggle lighting modes (Producers) instantly.
*   **Live Configuration:** Adjust colors, animation timing, and effect
    parameters without restarting the server.
*   **Unified UI:** Same codebase runs as a native Android app, a Linux
    desktop app, and a Web dashboard served directly by the GoLEDS
    server.
*   **Smart Safety:** Prevents data loss with atomic saves and validates
    configuration changes before applying them.

## Supported Platforms

*   **Android:** Full native APK with launcher integration.
*   **Web:** Progressive Web App (PWA) experience, typically hosted by the
    GoLEDS device itself.
*   **Linux:** Native desktop application.

## Development

This project follows standard Flutter conventions.

### Prerequisites

*   **Flutter SDK:** (Version matching `environment.sdk` in `pubspec.yaml`,
    currently `^3.10.4`)
*   **GoLEDS Server:** A running instance of the GoLEDS server (either on
    dedicated hardware like a Raspberry Pi or locally) to act as the API backend.

### Running Locally

1.  **Start the Backend:** Ensure your GoLEDS server is running. By
    default, it serves at `http://localhost:8080`.

    ```bash
    # From the project root
    go run .
    ```

2.  **Run the App:**
    ```bash
    cd mobile
    flutter run -d chrome  # For Web
    # OR
    flutter run -d linux   # For Desktop
    ```

    *Tip: The app automatically attempts to connect to
    `http://goleds.local:8080` or the local host. You can configure the
    server URL in the settings dialog.*

### Building

We use [Task](https://taskfile.dev/) (defined in the project root) to automate builds and asset
generation.

Build for Web:**
Builds the PWA and deploys it to the `web/` directory for the Go server to serve.
```bash
task build-web
```

**Build for Android:**
Generates `goleds_commander.apk`.
```bash
task build-android
```

**Build for Linux:**
Generates the desktop bundle in `commander_linux/`.
```bash
task build-linux
```

## Architecture

*   **State Management:** Uses `Provider` for reactive state updates.
*   **API:** Communicates with the Go backend via a REST-like API
    (`GET/POST /api/config`).
*   **Assets:**
    *   `assets/`: Build-time resources (Source icons).
    *   `images/`: Runtime assets (Backgrounds for producer cards).
