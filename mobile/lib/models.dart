const int _nsPerMs = 1000000;
const int _nsPerSec = 1000000000;

enum Producer { sensor, night, clock, audio, cylon, multiBlob }

class RuntimeConfig {
  final int ledsTotal;
  final SensorLEDConfig sensorLED;
  final NightLEDConfig nightLED;
  final ClockLEDConfig clockLED;
  final AudioLEDConfig audioLED;
  final CylonLEDConfig cylonLED;
  final MultiBlobLEDConfig multiBlobLED;

  const RuntimeConfig({
    required this.ledsTotal,
    required this.sensorLED,
    required this.nightLED,
    required this.clockLED,
    required this.audioLED,
    required this.cylonLED,
    required this.multiBlobLED,
  });

  RuntimeConfig copyWith({
    int? ledsTotal,
    SensorLEDConfig? sensorLED,
    NightLEDConfig? nightLED,
    ClockLEDConfig? clockLED,
    AudioLEDConfig? audioLED,
    CylonLEDConfig? cylonLED,
    MultiBlobLEDConfig? multiBlobLED,
  }) {
    return RuntimeConfig(
      ledsTotal: ledsTotal ?? this.ledsTotal,
      sensorLED: sensorLED ?? this.sensorLED,
      nightLED: nightLED ?? this.nightLED,
      clockLED: clockLED ?? this.clockLED,
      audioLED: audioLED ?? this.audioLED,
      cylonLED: cylonLED ?? this.cylonLED,
      multiBlobLED: multiBlobLED ?? this.multiBlobLED,
    );
  }

  RuntimeConfig toggleProducer(Producer producer, bool isEnabled) {
    switch (producer) {
      case Producer.sensor:
        return copyWith(
          sensorLED: sensorLED.copyWith(enabled: isEnabled),
          cylonLED: isEnabled ? cylonLED : cylonLED.copyWith(enabled: false),
          multiBlobLED: isEnabled
              ? multiBlobLED
              : multiBlobLED.copyWith(enabled: false),
        );
      case Producer.night:
        return copyWith(nightLED: nightLED.copyWith(enabled: isEnabled));
      case Producer.clock:
        return copyWith(clockLED: clockLED.copyWith(enabled: isEnabled));
      case Producer.audio:
        return copyWith(audioLED: audioLED.copyWith(enabled: isEnabled));
      case Producer.cylon:
        return copyWith(cylonLED: cylonLED.copyWith(enabled: isEnabled));
      case Producer.multiBlob:
        return copyWith(
          multiBlobLED: multiBlobLED.copyWith(enabled: isEnabled),
        );
    }
  }

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
  final bool enabled;
  final int runUpDelayMs;
  final int runDownDelayMs;
  final int holdTimeSec;
  final List<double> ledRGB;
  final bool latchEnabled;
  final int latchTriggerValue;
  final int latchTriggerDelaySec;
  final int latchTimeSec;
  final List<double> latchLedRGB;

  const SensorLEDConfig({
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

  SensorLEDConfig copyWith({
    bool? enabled,
    int? runUpDelayMs,
    int? runDownDelayMs,
    int? holdTimeSec,
    List<double>? ledRGB,
    bool? latchEnabled,
    int? latchTriggerValue,
    int? latchTriggerDelaySec,
    int? latchTimeSec,
    List<double>? latchLedRGB,
  }) {
    return SensorLEDConfig(
      enabled: enabled ?? this.enabled,
      runUpDelayMs: runUpDelayMs ?? this.runUpDelayMs,
      runDownDelayMs: runDownDelayMs ?? this.runDownDelayMs,
      holdTimeSec: holdTimeSec ?? this.holdTimeSec,
      ledRGB: ledRGB ?? List.from(this.ledRGB),
      latchEnabled: latchEnabled ?? this.latchEnabled,
      latchTriggerValue: latchTriggerValue ?? this.latchTriggerValue,
      latchTriggerDelaySec: latchTriggerDelaySec ?? this.latchTriggerDelaySec,
      latchTimeSec: latchTimeSec ?? this.latchTimeSec,
      latchLedRGB: latchLedRGB ?? List.from(this.latchLedRGB),
    );
  }

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
      'RunUpDelay': runUpDelayMs * _nsPerMs,
      'RunDownDelay': runDownDelayMs * _nsPerMs,
      'HoldTime': holdTimeSec * _nsPerSec,
      'LedRGB': ledRGB,
      'LatchEnabled': latchEnabled,
      'LatchTriggerValue': latchTriggerValue,
      'LatchTriggerDelay': latchTriggerDelaySec * _nsPerSec,
      'LatchTime': latchTimeSec * _nsPerSec,
      'LatchLedRGB': latchLedRGB,
    };
  }
}

class NightLEDConfig {
  final bool enabled;
  final double latitude;
  final double longitude;
  final List<List<double>> ledRGB;

  const NightLEDConfig({
    required this.enabled,
    required this.latitude,
    required this.longitude,
    required this.ledRGB,
  });

  NightLEDConfig copyWith({
    bool? enabled,
    double? latitude,
    double? longitude,
    List<List<double>>? ledRGB,
  }) {
    return NightLEDConfig(
      enabled: enabled ?? this.enabled,
      latitude: latitude ?? this.latitude,
      longitude: longitude ?? this.longitude,
      ledRGB: ledRGB ?? this.ledRGB.map((e) => List<double>.from(e)).toList(),
    );
  }

  factory NightLEDConfig.fromJson(Map<String, dynamic> json) {
    final list = json['LedRGB'] as List?;
    List<List<double>> rgbList = [];
    if (list != null) {
      rgbList = list.map((e) => _parseDoubleList(e)).toList();
    } else {
      rgbList = [
        [0.0, 0.0, 0.0],
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
  final bool enabled;
  final int startLedHour;
  final int endLedHour;
  final int startLedMinute;
  final int endLedMinute;
  final List<double> ledHour;
  final List<double> ledMinute;

  const ClockLEDConfig({
    required this.enabled,
    required this.startLedHour,
    required this.endLedHour,
    required this.startLedMinute,
    required this.endLedMinute,
    required this.ledHour,
    required this.ledMinute,
  });

  ClockLEDConfig copyWith({
    bool? enabled,
    int? startLedHour,
    int? endLedHour,
    int? startLedMinute,
    int? endLedMinute,
    List<double>? ledHour,
    List<double>? ledMinute,
  }) {
    return ClockLEDConfig(
      enabled: enabled ?? this.enabled,
      startLedHour: startLedHour ?? this.startLedHour,
      endLedHour: endLedHour ?? this.endLedHour,
      startLedMinute: startLedMinute ?? this.startLedMinute,
      endLedMinute: endLedMinute ?? this.endLedMinute,
      ledHour: ledHour ?? List.from(this.ledHour),
      ledMinute: ledMinute ?? List.from(this.ledMinute),
    );
  }

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

class SqueezeboxConfig {
  final String server;
  final int slimProtoPort;
  final int jsonrpcPort;
  final String playerMAC;
  final String playerName;
  final List<String> ignoredPlayers;
  final bool autoSync;
  final int pollIntervalMs;

  const SqueezeboxConfig({
    required this.server,
    required this.slimProtoPort,
    required this.jsonrpcPort,
    required this.playerMAC,
    required this.playerName,
    required this.ignoredPlayers,
    required this.autoSync,
    required this.pollIntervalMs,
  });

  SqueezeboxConfig copyWith({
    String? server,
    int? slimProtoPort,
    int? jsonrpcPort,
    String? playerMAC,
    String? playerName,
    List<String>? ignoredPlayers,
    bool? autoSync,
    int? pollIntervalMs,
  }) {
    return SqueezeboxConfig(
      server: server ?? this.server,
      slimProtoPort: slimProtoPort ?? this.slimProtoPort,
      jsonrpcPort: jsonrpcPort ?? this.jsonrpcPort,
      playerMAC: playerMAC ?? this.playerMAC,
      playerName: playerName ?? this.playerName,
      ignoredPlayers: ignoredPlayers ?? List.from(this.ignoredPlayers),
      autoSync: autoSync ?? this.autoSync,
      pollIntervalMs: pollIntervalMs ?? this.pollIntervalMs,
    );
  }

  factory SqueezeboxConfig.fromJson(Map<String, dynamic> json) {
    return SqueezeboxConfig(
      server: json['Server'] ?? '',
      slimProtoPort: json['SlimProtoPort'] ?? 3483,
      jsonrpcPort: json['JSONRPCPort'] ?? 9000,
      playerMAC: json['PlayerMAC'] ?? '',
      playerName: json['PlayerName'] ?? '',
      ignoredPlayers:
          (json['IgnoredPlayers'] as List?)
              ?.map((e) => e.toString())
              .toList() ??
          [],
      autoSync: json['AutoSync'] ?? false,
      pollIntervalMs: _parseDurationToMs(json['PollInterval']),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'Server': server,
      'SlimProtoPort': slimProtoPort,
      'JSONRPCPort': jsonrpcPort,
      'PlayerMAC': playerMAC,
      'PlayerName': playerName,
      'IgnoredPlayers': ignoredPlayers,
      'AutoSync': autoSync,
      'PollInterval': pollIntervalMs * _nsPerMs,
    };
  }
}

class AudioLEDConfig {
  final bool enabled;
  final int startLedLeft;
  final int endLedLeft;
  final int startLedRight;
  final int endLedRight;
  final List<double> ledGreen;
  final List<double> ledYellow;
  final List<double> ledRed;
  final bool peakHoldEnabled;
  final int peakHoldTimeMs;
  final double peakDecayRate;
  final int updateFreqMs;
  final double minDB;
  final double maxDB;
  final SqueezeboxConfig squeezebox;

  const AudioLEDConfig({
    required this.enabled,
    required this.startLedLeft,
    required this.endLedLeft,
    required this.startLedRight,
    required this.endLedRight,
    required this.ledGreen,
    required this.ledYellow,
    required this.ledRed,
    this.peakHoldEnabled = true,
    this.peakHoldTimeMs = 60,
    this.peakDecayRate = 20.0,
    required this.updateFreqMs,
    required this.minDB,
    required this.maxDB,
    required this.squeezebox,
  });

  AudioLEDConfig copyWith({
    bool? enabled,
    int? startLedLeft,
    int? endLedLeft,
    int? startLedRight,
    int? endLedRight,
    List<double>? ledGreen,
    List<double>? ledYellow,
    List<double>? ledRed,
    bool? peakHoldEnabled,
    int? peakHoldTimeMs,
    double? peakDecayRate,
    int? updateFreqMs,
    double? minDB,
    double? maxDB,
    SqueezeboxConfig? squeezebox,
  }) {
    return AudioLEDConfig(
      enabled: enabled ?? this.enabled,
      startLedLeft: startLedLeft ?? this.startLedLeft,
      endLedLeft: endLedLeft ?? this.endLedLeft,
      startLedRight: startLedRight ?? this.startLedRight,
      endLedRight: endLedRight ?? this.endLedRight,
      ledGreen: ledGreen ?? List.from(this.ledGreen),
      ledYellow: ledYellow ?? List.from(this.ledYellow),
      ledRed: ledRed ?? List.from(this.ledRed),
      peakHoldEnabled: peakHoldEnabled ?? this.peakHoldEnabled,
      peakHoldTimeMs: peakHoldTimeMs ?? this.peakHoldTimeMs,
      peakDecayRate: peakDecayRate ?? this.peakDecayRate,
      updateFreqMs: updateFreqMs ?? this.updateFreqMs,
      minDB: minDB ?? this.minDB,
      maxDB: maxDB ?? this.maxDB,
      squeezebox: squeezebox ?? this.squeezebox,
    );
  }

  factory AudioLEDConfig.fromJson(Map<String, dynamic> json) {
    return AudioLEDConfig(
      enabled: json['Enabled'] ?? false,
      startLedLeft: json['StartLedLeft'] ?? 0,
      endLedLeft: json['EndLedLeft'] ?? 0,
      startLedRight: json['StartLedRight'] ?? 0,
      endLedRight: json['EndLedRight'] ?? 0,
      ledGreen: _parseDoubleList(json['LedGreen']),
      ledYellow: _parseDoubleList(json['LedYellow']),
      ledRed: _parseDoubleList(json['LedRed']),
      peakHoldEnabled: json['PeakHoldEnabled'] ?? true,
      peakHoldTimeMs: json['PeakHoldTime'] != null
          ? _parseDurationToMs(json['PeakHoldTime'])
          : 250,
      peakDecayRate: (json['PeakDecayRate'] ?? 20.0).toDouble(),
      updateFreqMs: _parseDurationToMs(json['UpdateFreq']),
      minDB: (json['MinDB'] ?? -60.0).toDouble(),
      maxDB: (json['MaxDB'] ?? -3.0).toDouble(),
      squeezebox: SqueezeboxConfig.fromJson(json['Squeezebox'] ?? {}),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'Enabled': enabled,
      'StartLedLeft': startLedLeft,
      'EndLedLeft': endLedLeft,
      'StartLedRight': startLedRight,
      'EndLedRight': endLedRight,
      'LedGreen': ledGreen,
      'LedYellow': ledYellow,
      'LedRed': ledRed,
      'PeakHoldEnabled': peakHoldEnabled,
      'PeakHoldTime': peakHoldTimeMs * _nsPerMs,
      'PeakDecayRate': peakDecayRate,
      'UpdateFreq': updateFreqMs * _nsPerMs,
      'MinDB': minDB,
      'MaxDB': maxDB,
      'Squeezebox': squeezebox.toJson(),
    };
  }
}

class CylonLEDConfig {
  final bool enabled;
  final int durationSec;
  final int delayMs;
  final double step;
  final int width;
  final List<double> ledRGB;

  const CylonLEDConfig({
    required this.enabled,
    required this.durationSec,
    required this.delayMs,
    required this.step,
    required this.width,
    required this.ledRGB,
  });

  CylonLEDConfig copyWith({
    bool? enabled,
    int? durationSec,
    int? delayMs,
    double? step,
    int? width,
    List<double>? ledRGB,
  }) {
    return CylonLEDConfig(
      enabled: enabled ?? this.enabled,
      durationSec: durationSec ?? this.durationSec,
      delayMs: delayMs ?? this.delayMs,
      step: step ?? this.step,
      width: width ?? this.width,
      ledRGB: ledRGB ?? List.from(this.ledRGB),
    );
  }

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
      'Duration': durationSec * _nsPerSec,
      'Delay': delayMs * _nsPerMs,
      'Step': step,
      'Width': width,
      'LedRGB': ledRGB,
    };
  }
}

class MultiBlobLEDConfig {
  final bool enabled;
  final int durationSec;
  final int delayMs;
  final List<BlobCfg> blobCfg;

  const MultiBlobLEDConfig({
    required this.enabled,
    required this.durationSec,
    required this.delayMs,
    required this.blobCfg,
  });

  MultiBlobLEDConfig copyWith({
    bool? enabled,
    int? durationSec,
    int? delayMs,
    List<BlobCfg>? blobCfg,
  }) {
    return MultiBlobLEDConfig(
      enabled: enabled ?? this.enabled,
      durationSec: durationSec ?? this.durationSec,
      delayMs: delayMs ?? this.delayMs,
      blobCfg: blobCfg ?? this.blobCfg.map((b) => b.copyWith()).toList(),
    );
  }

  factory MultiBlobLEDConfig.fromJson(Map<String, dynamic> json) {
    final list = json['BlobCfg'] as List?;
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
      'Duration': durationSec * _nsPerSec,
      'Delay': delayMs * _nsPerMs,
      'BlobCfg': blobCfg.map((e) => e.toJson()).toList(),
    };
  }
}

class BlobCfg {
  final double deltaX;
  final double x;
  final double width;
  final List<double> ledRGB;

  const BlobCfg({
    required this.deltaX,
    required this.x,
    required this.width,
    required this.ledRGB,
  });

  BlobCfg copyWith({
    double? deltaX,
    double? x,
    double? width,
    List<double>? ledRGB,
  }) {
    return BlobCfg(
      deltaX: deltaX ?? this.deltaX,
      x: x ?? this.x,
      width: width ?? this.width,
      ledRGB: ledRGB ?? List.from(this.ledRGB),
    );
  }

  factory BlobCfg.fromJson(Map<String, dynamic> json) {
    return BlobCfg(
      deltaX: (json['DeltaX'] ?? 0).toDouble(),
      x: (json['X'] ?? 0).toDouble(),
      width: (json['Width'] ?? 0).toDouble(),
      ledRGB: _parseDoubleList(json['LedRGB']),
    );
  }

  Map<String, dynamic> toJson() {
    return {'DeltaX': deltaX, 'X': x, 'Width': width, 'LedRGB': ledRGB};
  }
}

List<double> _parseDoubleList(dynamic json) {
  if (json == null) return [0.0, 0.0, 0.0];
  if (json is List) {
    return json.map((e) => (e as num).toDouble()).toList();
  }
  return [0.0, 0.0, 0.0];
}

int _parseDurationToMs(dynamic val) {
  if (val == null) return 0;
  if (val is num) {
    return (val / _nsPerMs).round();
  }
  return 0;
}

int _parseDurationToSec(dynamic val) {
  if (val == null) return 0;
  if (val is num) {
    return (val / _nsPerSec).round();
  }
  return 0;
}
