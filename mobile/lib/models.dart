

class RuntimeConfig {
  int ledsTotal;
  SensorLEDConfig sensorLED;
  NightLEDConfig nightLED;
  ClockLEDConfig clockLED;
  AudioLEDConfig audioLED;
  CylonLEDConfig cylonLED;
  MultiBlobLEDConfig multiBlobLED;

  RuntimeConfig({
    required this.ledsTotal,
    required this.sensorLED,
    required this.nightLED,
    required this.clockLED,
    required this.audioLED,
    required this.cylonLED,
    required this.multiBlobLED,
  });

  factory RuntimeConfig.fromJson(Map<String, dynamic> json) {
    return RuntimeConfig(
      ledsTotal: json['LedsTotal'] ?? 0,
      sensorLED: SensorLEDConfig.fromJson(json['SensorLED'] ?? {}),
      nightLED: NightLEDConfig.fromJson(json['NightLED'] ?? {}),
      clockLED: ClockLEDConfig.fromJson(json['ClockLED'] ?? {}),
      audioLED: AudioLEDConfig.fromJson(json['AudioLED'] ?? {}),
      cylonLED: CylonLEDConfig.fromJson(json['CylonLED'] ?? {}),
      multiBlobLED: MultiBlobLEDConfig.fromJson(json['MultiBlobLED'] ?? {}),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'LedsTotal': ledsTotal,
      'SensorLED': sensorLED.toJson(),
      'NightLED': nightLED.toJson(),
      'ClockLED': clockLED.toJson(),
      'AudioLED': audioLED.toJson(),
      'CylonLED': cylonLED.toJson(),
      'MultiBlobLED': multiBlobLED.toJson(),
    };
  }
}

class SensorLEDConfig {
  bool enabled;
  int runUpDelayMs;
  int runDownDelayMs;
  int holdTimeSec;
  List<double> ledRGB;
  bool latchEnabled;
  int latchTriggerValue;
  int latchTriggerDelaySec;
  int latchTimeSec;
  List<double> latchLedRGB;

  SensorLEDConfig({
    required this.enabled,
    required this.runUpDelayMs,
    required this.runDownDelayMs,
    required this.holdTimeSec,
    required this.ledRGB,
    required this.latchEnabled,
    required this.latchTriggerValue,
    required this.latchTriggerDelaySec,
    required this.latchTimeSec,
    required this.latchLedRGB,
  });

  factory SensorLEDConfig.fromJson(Map<String, dynamic> json) {
    return SensorLEDConfig(
      enabled: json['Enabled'] ?? false,
      runUpDelayMs: _parseDurationToMs(json['RunUpDelay']),
      runDownDelayMs: _parseDurationToMs(json['RunDownDelay']),
      holdTimeSec: _parseDurationToSec(json['HoldTime']),
      ledRGB: _parseDoubleList(json['LedRGB']),
      latchEnabled: json['LatchEnabled'] ?? false,
      latchTriggerValue: json['LatchTriggerValue'] ?? 0,
      latchTriggerDelaySec: _parseDurationToSec(json['LatchTriggerDelay']),
      latchTimeSec: _parseDurationToSec(json['LatchTime']),
      latchLedRGB: _parseDoubleList(json['LatchLedRGB']),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'Enabled': enabled,
      'RunUpDelay': runUpDelayMs * 1000000,
      'RunDownDelay': runDownDelayMs * 1000000,
      'HoldTime': holdTimeSec * 1000000000,
      'LedRGB': ledRGB,
      'LatchEnabled': latchEnabled,
      'LatchTriggerValue': latchTriggerValue,
      'LatchTriggerDelay': latchTriggerDelaySec * 1000000000,
      'LatchTime': latchTimeSec * 1000000000,
      'LatchLedRGB': latchLedRGB,
    };
  }
}

class NightLEDConfig {
  bool enabled;
  double latitude;
  double longitude;
  List<List<double>> ledRGB;

  NightLEDConfig({
    required this.enabled,
    required this.latitude,
    required this.longitude,
    required this.ledRGB,
  });

  factory NightLEDConfig.fromJson(Map<String, dynamic> json) {
    var list = json['LedRGB'] as List?;
    List<List<double>> rgbList = [];
    if (list != null) {
      rgbList = list.map((e) => _parseDoubleList(e)).toList();
    } else {
      rgbList = [
        [0.0, 0.0, 0.0]
      ];
    }
    return NightLEDConfig(
      enabled: json['Enabled'] ?? false,
      latitude: (json['Latitude'] ?? 0).toDouble(),
      longitude: (json['Longitude'] ?? 0).toDouble(),
      ledRGB: rgbList,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'Enabled': enabled,
      'Latitude': latitude,
      'Longitude': longitude,
      'LedRGB': ledRGB,
    };
  }
}

class ClockLEDConfig {
  bool enabled;
  int startLedHour;
  int endLedHour;
  int startLedMinute;
  int endLedMinute;
  List<double> ledHour;
  List<double> ledMinute;

  ClockLEDConfig({
    required this.enabled,
    required this.startLedHour,
    required this.endLedHour,
    required this.startLedMinute,
    required this.endLedMinute,
    required this.ledHour,
    required this.ledMinute,
  });

  factory ClockLEDConfig.fromJson(Map<String, dynamic> json) {
    return ClockLEDConfig(
      enabled: json['Enabled'] ?? false,
      startLedHour: json['StartLedHour'] ?? 0,
      endLedHour: json['EndLedHour'] ?? 0,
      startLedMinute: json['StartLedMinute'] ?? 0,
      endLedMinute: json['EndLedMinute'] ?? 0,
      ledHour: _parseDoubleList(json['LedHour']),
      ledMinute: _parseDoubleList(json['LedMinute']),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'Enabled': enabled,
      'StartLedHour': startLedHour,
      'EndLedHour': endLedHour,
      'StartLedMinute': startLedMinute,
      'EndLedMinute': endLedMinute,
      'LedHour': ledHour,
      'LedMinute': ledMinute,
    };
  }
}

class AudioLEDConfig {
  bool enabled;
  String device;
  int startLedLeft;
  int endLedLeft;
  int startLedRight;
  int endLedRight;
  List<double> ledGreen;
  List<double> ledYellow;
  List<double> ledRed;
  int sampleRate;
  int framesPerBuffer;
  int updateFreqMs;
  double minDB;
  double maxDB;

  AudioLEDConfig({
    required this.enabled,
    required this.device,
    required this.startLedLeft,
    required this.endLedLeft,
    required this.startLedRight,
    required this.endLedRight,
    required this.ledGreen,
    required this.ledYellow,
    required this.ledRed,
    required this.sampleRate,
    required this.framesPerBuffer,
    required this.updateFreqMs,
    required this.minDB,
    required this.maxDB,
  });

  factory AudioLEDConfig.fromJson(Map<String, dynamic> json) {
    return AudioLEDConfig(
      enabled: json['Enabled'] ?? false,
      device: json['Device'] ?? '',
      startLedLeft: json['StartLedLeft'] ?? 0,
      endLedLeft: json['EndLedLeft'] ?? 0,
      startLedRight: json['StartLedRight'] ?? 0,
      endLedRight: json['EndLedRight'] ?? 0,
      ledGreen: _parseDoubleList(json['LedGreen']),
      ledYellow: _parseDoubleList(json['LedYellow']),
      ledRed: _parseDoubleList(json['LedRed']),
      sampleRate: json['SampleRate'] ?? 0,
      framesPerBuffer: json['FramesPerBuffer'] ?? 0,
      updateFreqMs: _parseDurationToMs(json['UpdateFreq']),
      minDB: (json['MinDB'] ?? 0).toDouble(),
      maxDB: (json['MaxDB'] ?? 0).toDouble(),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'Enabled': enabled,
      'Device': device,
      'StartLedLeft': startLedLeft,
      'EndLedLeft': endLedLeft,
      'StartLedRight': startLedRight,
      'EndLedRight': endLedRight,
      'LedGreen': ledGreen,
      'LedYellow': ledYellow,
      'LedRed': ledRed,
      'SampleRate': sampleRate,
      'FramesPerBuffer': framesPerBuffer,
      'UpdateFreq': updateFreqMs * 1000000,
      'MinDB': minDB,
      'MaxDB': maxDB,
    };
  }
}

class CylonLEDConfig {
  bool enabled;
  int durationSec;
  int delayMs;
  double step;
  int width;
  List<double> ledRGB;

  CylonLEDConfig({
    required this.enabled,
    required this.durationSec,
    required this.delayMs,
    required this.step,
    required this.width,
    required this.ledRGB,
  });

  factory CylonLEDConfig.fromJson(Map<String, dynamic> json) {
    return CylonLEDConfig(
      enabled: json['Enabled'] ?? false,
      durationSec: _parseDurationToSec(json['Duration']),
      delayMs: _parseDurationToMs(json['Delay']),
      step: (json['Step'] ?? 0).toDouble(),
      width: json['Width'] ?? 0,
      ledRGB: _parseDoubleList(json['LedRGB']),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'Enabled': enabled,
      'Duration': durationSec * 1000000000,
      'Delay': delayMs * 1000000,
      'Step': step,
      'Width': width,
      'LedRGB': ledRGB,
    };
  }
}

class MultiBlobLEDConfig {
  bool enabled;
  int durationSec;
  int delayMs;
  List<BlobCfg> blobCfg;

  MultiBlobLEDConfig({
    required this.enabled,
    required this.durationSec,
    required this.delayMs,
    required this.blobCfg,
  });

  factory MultiBlobLEDConfig.fromJson(Map<String, dynamic> json) {
    var list = json['BlobCfg'] as List?;
    List<BlobCfg> blobs = [];
    if (list != null) {
      blobs = list.map((e) => BlobCfg.fromJson(e)).toList();
    }
    return MultiBlobLEDConfig(
      enabled: json['Enabled'] ?? false,
      durationSec: _parseDurationToSec(json['Duration']),
      delayMs: _parseDurationToMs(json['Delay']),
      blobCfg: blobs,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'Enabled': enabled,
      'Duration': durationSec * 1000000000,
      'Delay': delayMs * 1000000,
      'BlobCfg': blobCfg.map((e) => e.toJson()).toList(),
    };
  }
}

class BlobCfg {
  double deltaX;
  double x;
  double width;
  List<double> ledRGB;

  BlobCfg({
    required this.deltaX,
    required this.x,
    required this.width,
    required this.ledRGB,
  });

  factory BlobCfg.fromJson(Map<String, dynamic> json) {
    return BlobCfg(
      deltaX: (json['DeltaX'] ?? 0).toDouble(),
      x: (json['X'] ?? 0).toDouble(),
      width: (json['Width'] ?? 0).toDouble(),
      ledRGB: _parseDoubleList(json['LedRGB']),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'DeltaX': deltaX,
      'X': x,
      'Width': width,
      'LedRGB': ledRGB,
    };
  }
}

// Helper functions for parsing

List<double> _parseDoubleList(dynamic json) {
  if (json == null) return [0.0, 0.0, 0.0];
  if (json is List) {
    return json.map((e) => (e as num).toDouble()).toList();
  }
  return [0.0, 0.0, 0.0];
}

int _parseDurationToMs(dynamic val) {
  if (val == null) return 0;
  // This assumes the Go server sends the value in nanoseconds (which standard Go json encoding for Duration does)
  // OR as a string like "10ms".
  // Go's standard json.Marshal encodes time.Duration as int64 nanoseconds.
  // HOWEVER, go-yaml might be involved or custom marshalling.
  // BUT the webhandler in Go reads using yaml but writes using json.
  // Standard json.Marshal of time.Duration is nanoseconds.
  // Wait, in `config.go`, structs have `yaml` tags.
  // In `webhandler.go`: `json.NewEncoder(w).Encode(runtimeConfig)`.
  // Standard Go `json` package encodes `time.Duration` as **nanoseconds** (integer).
  // BUT the `yaml.v3` decoder reads strings like "10ms".
  // The user sees "10ms" in config.yml.
  // Let's verify what the Go server sends.
  // Standard library `json` encodes duration as integer (nanoseconds).
  //
  // IF the struct fields are just `time.Duration`, JSON output is `10000000` for 10ms.
  //
  // Let's assume nanoseconds.
  if (val is num) {
    return (val / 1000000).round();
  }
  return 0;
}

int _parseDurationToSec(dynamic val) {
  if (val == null) return 0;
  if (val is num) {
    return (val / 1000000000).round();
  }
  return 0;
}
