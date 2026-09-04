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
  late SqueezeboxConfig squeezeboxConfig;

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
    squeezeboxConfig = a.squeezebox;

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

  void _save() {
    final provider = context.read<ConfigProvider>();
    final currentFullConfig = provider.config;
    if (currentFullConfig == null) return;

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
      squeezebox: squeezeboxConfig,
    );

    provider
        .updateConfig(currentFullConfig.copyWith(audioLED: updatedAudioConfig))
        .then((_) {
          if (mounted) Navigator.pop(context);
        });
  }

  void _openSqueezeboxDialog() async {
    final updated = await showDialog<SqueezeboxConfig>(
      context: context,
      builder: (ctx) =>
          _SqueezeboxConfigDialog(initialConfig: squeezeboxConfig),
    );
    if (updated != null) {
      setState(() {
        squeezeboxConfig = updated;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Audio VU Config'),
        actions: [
          IconButton(
            icon: const Icon(Icons.speaker_group),
            tooltip: 'Squeezebox Settings',
            onPressed: _openSqueezeboxDialog,
          ),
          IconButton(icon: const Icon(Icons.save), onPressed: _save),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            color: const Color(0xFF1E1E1E),
            margin: const EdgeInsets.only(bottom: 24),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8),
              side: BorderSide(
                color: Colors.greenAccent.withValues(alpha: 0.3),
              ),
            ),
            child: ListTile(
              leading: const Icon(
                Icons.speaker_group,
                color: Colors.greenAccent,
              ),
              title: const Text(
                'Squeezebox / LMS Server',
                style: TextStyle(fontWeight: FontWeight.bold),
              ),
              subtitle: Text(
                squeezeboxConfig.server.isEmpty
                    ? 'Auto-discovery • ${squeezeboxConfig.playerName.isEmpty ? "GoLEDs VU" : squeezeboxConfig.playerName}'
                    : '${squeezeboxConfig.server} • ${squeezeboxConfig.playerName.isEmpty ? "GoLEDs VU" : squeezeboxConfig.playerName}',
                style: const TextStyle(color: Colors.grey),
              ),
              trailing: const Icon(Icons.tune, color: Colors.greenAccent),
              onTap: _openSqueezeboxDialog,
            ),
          ),
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

class _SqueezeboxConfigDialog extends StatefulWidget {
  final SqueezeboxConfig initialConfig;

  const _SqueezeboxConfigDialog({required this.initialConfig});

  @override
  State<_SqueezeboxConfigDialog> createState() =>
      _SqueezeboxConfigDialogState();
}

class _SqueezeboxConfigDialogState extends State<_SqueezeboxConfigDialog> {
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

  @override
  void initState() {
    super.initState();
    final s = widget.initialConfig;

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

  void _submit() {
    final ignoredText = ignoredPlayersCtrl.text.trim();
    final ignoredPlayers = ignoredText.isEmpty
        ? <String>[]
        : ignoredText
              .split(',')
              .map((e) => e.trim())
              .where((e) => e.isNotEmpty)
              .toList();

    final result = SqueezeboxConfig(
      server: serverCtrl.text.trim(),
      slimProtoPort: int.tryParse(slimProtoPortCtrl.text.trim()) ?? 0,
      jsonrpcPort: int.tryParse(jsonrpcPortCtrl.text.trim()) ?? 0,
      playerName: playerNameCtrl.text.trim(),
      playerMAC: playerMACCtrl.text.trim(),
      ignoredPlayers: ignoredPlayers,
      autoSync: autoSync,
      pollIntervalMs: pollIntervalMs,
    );

    Navigator.pop(context, result);
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Row(
        children: [
          Icon(Icons.speaker_group, color: Colors.greenAccent),
          SizedBox(width: 8),
          Text('Squeezebox / LMS', style: TextStyle(fontSize: 18)),
        ],
      ),
      content: SizedBox(
        width: 400,
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
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
                        hintText: slimProtoPortFocusNode.hasFocus
                            ? '3483'
                            : null,
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
                contentPadding: EdgeInsets.zero,
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
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('Cancel'),
        ),
        ElevatedButton(onPressed: _submit, child: const Text('Done')),
      ],
    );
  }
}
