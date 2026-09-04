import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../widgets/color_picker_tile.dart';
import '../../widgets/config_slider.dart';
import '../../widgets/led_selectors.dart';
import '../../widgets/section_header.dart';
import '../../models.dart';
import '../../utils.dart';

class AudioLEDEditor extends StatefulWidget {
  final AudioLEDConfig initialConfig;
  final int totalLeds;

  const AudioLEDEditor({
    super.key,
    required this.initialConfig,
    required this.totalLeds,
  });

  @override
  State<AudioLEDEditor> createState() => _AudioLEDEditorState();
}

class _AudioLEDEditorState extends State<AudioLEDEditor> {
  // Squeezebox Controllers & FocusNodes
  late TextEditingController serverCtrl;
  late TextEditingController slimProtoPortCtrl;
  late TextEditingController jsonrpcPortCtrl;
  late TextEditingController playerNameCtrl;
  late TextEditingController playerMACCtrl;
  late TextEditingController ignoredPlayersCtrl;

  late FocusNode serverFocusNode;
  late FocusNode slimProtoPortFocusNode;
  late FocusNode jsonrpcPortFocusNode;
  late FocusNode playerNameFocusNode;
  late FocusNode playerMACFocusNode;
  late FocusNode ignoredPlayersFocusNode;

  late bool autoSync;
  late int pollIntervalMs;

  // VU Meter & Channel Settings
  late int startLedLeft, endLedLeft;
  late int startLedRight, endLedRight;
  late Color ledGreen, ledYellow, ledRed;
  late int updateFreqMs;
  late double minDB, maxDB;

  @override
  void initState() {
    super.initState();
    final a = widget.initialConfig;
    final s = a.squeezebox;

    serverCtrl = TextEditingController(text: s.server);
    slimProtoPortCtrl = TextEditingController(
      text: s.slimProtoPort > 0 ? s.slimProtoPort.toString() : '',
    );
    jsonrpcPortCtrl = TextEditingController(
      text: s.jsonrpcPort > 0 ? s.jsonrpcPort.toString() : '',
    );
    playerNameCtrl = TextEditingController(text: s.playerName);
    playerMACCtrl = TextEditingController(text: s.playerMAC);
    ignoredPlayersCtrl = TextEditingController(
      text: s.ignoredPlayers.join(', '),
    );

    serverFocusNode = FocusNode()..addListener(() => setState(() {}));
    slimProtoPortFocusNode = FocusNode()..addListener(() => setState(() {}));
    jsonrpcPortFocusNode = FocusNode()..addListener(() => setState(() {}));
    playerNameFocusNode = FocusNode()..addListener(() => setState(() {}));
    playerMACFocusNode = FocusNode()..addListener(() => setState(() {}));
    ignoredPlayersFocusNode = FocusNode()..addListener(() => setState(() {}));

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
  }

  @override
  void dispose() {
    serverCtrl.dispose();
    slimProtoPortCtrl.dispose();
    jsonrpcPortCtrl.dispose();
    playerNameCtrl.dispose();
    playerMACCtrl.dispose();
    ignoredPlayersCtrl.dispose();

    serverFocusNode.dispose();
    slimProtoPortFocusNode.dispose();
    jsonrpcPortFocusNode.dispose();
    playerNameFocusNode.dispose();
    playerMACFocusNode.dispose();
    ignoredPlayersFocusNode.dispose();

    super.dispose();
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final currentFullConfig = provider.config;
    if (currentFullConfig == null) return;

    final ignoredText = ignoredPlayersCtrl.text.trim();
    final ignoredPlayers = ignoredText.isEmpty
        ? <String>[]
        : ignoredText
              .split(',')
              .map((e) => e.trim())
              .where((e) => e.isNotEmpty)
              .toList();

    final updatedSqueezebox = currentFullConfig.audioLED.squeezebox.copyWith(
      server: serverCtrl.text.trim(),
      slimProtoPort: int.tryParse(slimProtoPortCtrl.text.trim()) ?? 0,
      jsonrpcPort: int.tryParse(jsonrpcPortCtrl.text.trim()) ?? 0,
      playerName: playerNameCtrl.text.trim(),
      playerMAC: playerMACCtrl.text.trim(),
      autoSync: autoSync,
      pollIntervalMs: pollIntervalMs,
      ignoredPlayers: ignoredPlayers,
    );

    final updatedAudioConfig = currentFullConfig.audioLED.copyWith(
      startLedLeft: startLedLeft,
      endLedLeft: endLedLeft,
      startLedRight: startLedRight,
      endLedRight: endLedRight,
      ledGreen: toRgbList(ledGreen),
      ledYellow: toRgbList(ledYellow),
      ledRed: toRgbList(ledRed),
      updateFreqMs: updateFreqMs,
      minDB: minDB,
      maxDB: maxDB,
      squeezebox: updatedSqueezebox,
    );

    provider
        .updateConfig(currentFullConfig.copyWith(audioLED: updatedAudioConfig))
        .then((_) {
          if (mounted) Navigator.pop(context);
        });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Audio VU Config'),
        actions: [IconButton(icon: const Icon(Icons.save), onPressed: _save)],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const SectionHeader(
            'Logitech Media Server (LMS)',
            color: Colors.greenAccent,
          ),
          TextField(
            controller: serverCtrl,
            focusNode: serverFocusNode,
            decoration: InputDecoration(
              labelText: 'LMS Server Host/IP',
              floatingLabelBehavior: FloatingLabelBehavior.always,
              hintText: serverFocusNode.hasFocus
                  ? 'Auto-discovery when empty'
                  : null,
              border: const OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: slimProtoPortCtrl,
                  focusNode: slimProtoPortFocusNode,
                  keyboardType: TextInputType.number,
                  decoration: InputDecoration(
                    labelText: 'SlimProto Port',
                    floatingLabelBehavior: FloatingLabelBehavior.always,
                    hintText: slimProtoPortFocusNode.hasFocus ? '3483' : null,
                    border: const OutlineInputBorder(),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: TextField(
                  controller: jsonrpcPortCtrl,
                  focusNode: jsonrpcPortFocusNode,
                  keyboardType: TextInputType.number,
                  decoration: InputDecoration(
                    labelText: 'JSON-RPC Port',
                    floatingLabelBehavior: FloatingLabelBehavior.always,
                    hintText: jsonrpcPortFocusNode.hasFocus ? '9000' : null,
                    border: const OutlineInputBorder(),
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          TextField(
            controller: playerNameCtrl,
            focusNode: playerNameFocusNode,
            decoration: InputDecoration(
              labelText: 'Player Name',
              floatingLabelBehavior: FloatingLabelBehavior.always,
              hintText: playerNameFocusNode.hasFocus ? 'GoLEDs VU' : null,
              border: const OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: playerMACCtrl,
            focusNode: playerMACFocusNode,
            decoration: InputDecoration(
              labelText: 'Player MAC Address',
              floatingLabelBehavior: FloatingLabelBehavior.always,
              hintText: playerMACFocusNode.hasFocus
                  ? 'Auto-generated when empty'
                  : null,
              border: const OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 16),
          SwitchListTile(
            title: const Text('Auto-Sync to Active Player'),
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
              focusNode: ignoredPlayersFocusNode,
              decoration: const InputDecoration(
                labelText: 'Ignored Players',
                floatingLabelBehavior: FloatingLabelBehavior.always,
                border: OutlineInputBorder(),
              ),
            ),
          ],
          const SizedBox(height: 24),
          const SectionHeader(
            'VU Meter Timing & Sensitivity',
            color: Colors.greenAccent,
          ),
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
          const SectionHeader('Channel Mapping', color: Colors.greenAccent),
          LedRangeSelector(
            label: 'Left Channel',
            start: startLedLeft,
            end: endLedLeft,
            totalLeds: widget.totalLeds,
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
            totalLeds: widget.totalLeds,
            onChanged: (s, e) => setState(() {
              startLedRight = s;
              endLedRight = e;
            }),
          ),
          const SizedBox(height: 24),
          const SectionHeader('VU Colors', color: Colors.greenAccent),
          ColorPickerTile(
            label: 'Low (Green)',
            color: ledGreen,
            onColorChanged: (c) => setState(() => ledGreen = c),
          ),
          ColorPickerTile(
            label: 'Mid (Yellow)',
            color: ledYellow,
            onColorChanged: (c) => setState(() => ledYellow = c),
          ),
          ColorPickerTile(
            label: 'High (Red)',
            color: ledRed,
            onColorChanged: (c) => setState(() => ledRed = c),
          ),
        ],
      ),
    );
  }
}
