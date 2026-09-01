import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../widgets/color_picker_tile.dart';
import '../../widgets/config_slider.dart';
import '../../widgets/led_selectors.dart';
import '../../utils.dart';

class AudioLEDEditor extends StatefulWidget {
  const AudioLEDEditor({super.key});

  @override
  State<AudioLEDEditor> createState() => _AudioLEDEditorState();
}

class _AudioLEDEditorState extends State<AudioLEDEditor> {
  // Squeezebox Controllers
  late TextEditingController serverCtrl;
  late TextEditingController slimProtoPortCtrl;
  late TextEditingController jsonrpcPortCtrl;
  late TextEditingController playerNameCtrl;
  late TextEditingController playerMACCtrl;
  late TextEditingController ignoredPlayersCtrl;
  late bool autoSync;
  late int pollIntervalMs;

  // VU Meter & Channel Settings
  late int startLedLeft, endLedLeft;
  late int startLedRight, endLedRight;
  late Color ledGreen, ledYellow, ledRed;
  late int updateFreqMs;
  late double minDB, maxDB;
  late int ledsTotal;

  bool _initialized = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_initialized) {
      final config = context.read<ConfigProvider>().config;
      if (config != null) {
        ledsTotal = config.ledsTotal;
        final a = config.audioLED;
        final s = a.squeezebox;

        serverCtrl = TextEditingController(text: s.server);
        slimProtoPortCtrl = TextEditingController(text: s.slimProtoPort.toString());
        jsonrpcPortCtrl = TextEditingController(text: s.jsonrpcPort.toString());
        playerNameCtrl = TextEditingController(text: s.playerName);
        playerMACCtrl = TextEditingController(text: s.playerMAC);
        ignoredPlayersCtrl = TextEditingController(text: s.ignoredPlayers.join(', '));
        autoSync = s.autoSync;
        pollIntervalMs = s.pollIntervalMs > 0 ? s.pollIntervalMs : 1500;

        startLedLeft = a.startLedLeft;
        endLedLeft = a.endLedLeft;
        startLedRight = a.startLedRight;
        endLedRight = a.endLedRight;
        ledGreen = fromRgbList(a.ledGreen);
        ledYellow = fromRgbList(a.ledYellow);
        ledRed = fromRgbList(a.ledRed);
        updateFreqMs = a.updateFreqMs > 0 ? a.updateFreqMs : 30;
        minDB = a.minDB;
        maxDB = a.maxDB;
        _initialized = true;
      }
    }
  }

  @override
  void dispose() {
    serverCtrl.dispose();
    slimProtoPortCtrl.dispose();
    jsonrpcPortCtrl.dispose();
    playerNameCtrl.dispose();
    playerMACCtrl.dispose();
    ignoredPlayersCtrl.dispose();
    super.dispose();
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final config = provider.config;
    if (config == null) return;

    final s = config.audioLED.squeezebox;
    s.server = serverCtrl.text.trim();
    s.slimProtoPort = int.tryParse(slimProtoPortCtrl.text.trim()) ?? 3483;
    s.jsonrpcPort = int.tryParse(jsonrpcPortCtrl.text.trim()) ?? 9000;
    s.playerName = playerNameCtrl.text.trim();
    s.playerMAC = playerMACCtrl.text.trim();
    s.autoSync = autoSync;
    s.pollIntervalMs = pollIntervalMs;

    final ignoredText = ignoredPlayersCtrl.text.trim();
    if (ignoredText.isEmpty) {
      s.ignoredPlayers = [];
    } else {
      s.ignoredPlayers = ignoredText
          .split(',')
          .map((e) => e.trim())
          .where((e) => e.isNotEmpty)
          .toList();
    }

    config.audioLED.startLedLeft = startLedLeft;
    config.audioLED.endLedLeft = endLedLeft;
    config.audioLED.startLedRight = startLedRight;
    config.audioLED.endLedRight = endLedRight;
    config.audioLED.ledGreen = toRgbList(ledGreen);
    config.audioLED.ledYellow = toRgbList(ledYellow);
    config.audioLED.ledRed = toRgbList(ledRed);
    config.audioLED.updateFreqMs = updateFreqMs;
    config.audioLED.minDB = minDB;
    config.audioLED.maxDB = maxDB;

    provider.updateConfig(config).then((_) {
      if (mounted) Navigator.pop(context);
    });
  }

  @override
  Widget build(BuildContext context) {
    if (!_initialized) return const Scaffold(body: Center(child: CircularProgressIndicator()));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Audio VU Config'),
        actions: [IconButton(icon: const Icon(Icons.save), onPressed: _save)],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSectionHeader('Logitech Media Server (LMS)'),
          TextField(
            controller: serverCtrl,
            decoration: const InputDecoration(
              labelText: 'LMS Server Host/IP',
              border: OutlineInputBorder(),
              helperText: 'e.g. 192.168.1.100 or 127.0.0.1',
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: slimProtoPortCtrl,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(
                    labelText: 'SlimProto Port',
                    border: OutlineInputBorder(),
                    helperText: 'Default: 3483',
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: TextField(
                  controller: jsonrpcPortCtrl,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(
                    labelText: 'JSON-RPC Port',
                    border: OutlineInputBorder(),
                    helperText: 'Default: 9000',
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          TextField(
            controller: playerNameCtrl,
            decoration: const InputDecoration(
              labelText: 'Player Name',
              border: OutlineInputBorder(),
              helperText: 'Name displayed in LMS (e.g. "GoLEDs VU")',
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: playerMACCtrl,
            decoration: const InputDecoration(
              labelText: 'Player MAC Address',
              border: OutlineInputBorder(),
              helperText: 'Optional: leave empty or "auto" to generate',
            ),
          ),
          const SizedBox(height: 16),
          SwitchListTile(
            title: const Text('Auto-Sync to Active Player'),
            subtitle: const Text('Automatically sync with any playing Squeezebox'),
            value: autoSync,
            onChanged: (val) => setState(() => autoSync = val),
            activeThumbColor: Colors.greenAccent,
          ),
          if (autoSync) ...[
            const SizedBox(height: 8),
            ConfigSlider(
              label: 'Sync Poll Interval',
              value: pollIntervalMs.toDouble(),
              min: 500,
              max: 5000,
              unit: 'ms',
              onChanged: (v) => setState(() => pollIntervalMs = v.toInt()),
              activeColor: Colors.greenAccent,
            ),
            const SizedBox(height: 12),
            TextField(
              controller: ignoredPlayersCtrl,
              decoration: const InputDecoration(
                labelText: 'Ignored Players',
                border: OutlineInputBorder(),
                helperText: 'Comma-separated player names or MACs to ignore',
              ),
            ),
          ],

          const SizedBox(height: 24),
          _buildSectionHeader('VU Meter Timing & Sensitivity'),
          ConfigSlider(
            label: 'LED Update Frequency',
            value: updateFreqMs.toDouble(),
            min: 10,
            max: 100,
            unit: 'ms',
            onChanged: (v) => setState(() => updateFreqMs = v.toInt()),
            activeColor: Colors.greenAccent,
          ),
          const SizedBox(height: 16),
          DbRangeSelector(
            label: 'Sensitivity Range (dB)',
            minDb: minDB,
            maxDb: maxDB,
            onChanged: (min, max) => setState(() {
              minDB = min;
              maxDB = max;
            }),
          ),

          const SizedBox(height: 24),
          _buildSectionHeader('Channel Mapping'),
          LedRangeSelector(
            label: 'Left Channel',
            start: startLedLeft,
            end: endLedLeft,
            totalLeds: ledsTotal,
            onChanged: (s, e) => setState(() {
              startLedLeft = s;
              endLedLeft = e;
            }),
          ),
          const SizedBox(height: 16),
          LedRangeSelector(
            label: 'Right Channel',
            start: startLedRight,
            end: endLedRight,
            totalLeds: ledsTotal,
            onChanged: (s, e) => setState(() {
              startLedRight = s;
              endLedRight = e;
            }),
          ),

          const SizedBox(height: 24),
          _buildSectionHeader('VU Colors'),
          ColorPickerTile(label: 'Low (Green)', color: ledGreen, onColorChanged: (c) => setState(() => ledGreen = c)),
          ColorPickerTile(label: 'Mid (Yellow)', color: ledYellow, onColorChanged: (c) => setState(() => ledYellow = c)),
          ColorPickerTile(label: 'High (Red)', color: ledRed, onColorChanged: (c) => setState(() => ledRed = c)),
        ],
      ),
    );
  }

  Widget _buildSectionHeader(String title) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Text(
        title.toUpperCase(),
        style: const TextStyle(
          color: Colors.greenAccent,
          fontWeight: FontWeight.bold,
          letterSpacing: 1.2,
        ),
      ),
    );
  }
}
