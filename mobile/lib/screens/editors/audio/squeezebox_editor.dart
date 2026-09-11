import 'package:flutter/material.dart';
import '../../../models.dart';
import '../../../widgets/config_slider.dart';
import '../../../widgets/section_header.dart';

class AudioSqueezeboxEditor extends StatefulWidget {
  final SqueezeboxConfig initialConfig;

  const AudioSqueezeboxEditor({
    super.key,
    required this.initialConfig,
  });

  @override
  State<AudioSqueezeboxEditor> createState() => _AudioSqueezeboxEditorState();
}

class _AudioSqueezeboxEditorState extends State<AudioSqueezeboxEditor> {
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

  void _done() {
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
    return Scaffold(
      appBar: AppBar(
        title: const Text('Squeezebox / LMS Server'),
        actions: [
          IconButton(
            icon: const Icon(Icons.check),
            tooltip: 'Apply',
            onPressed: _done,
          ),
        ],
      ),
      body: PopScope(
        canPop: false,
        onPopInvokedWithResult: (didPop, result) {
          if (didPop) return;
          _done();
        },
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            const SectionHeader('Server Connection', color: Colors.greenAccent),
            TextField(
              controller: serverCtrl,
              focusNode: serverFocusNode,
              decoration: InputDecoration(
                labelText: 'LMS Server Host / IP',
                floatingLabelBehavior: FloatingLabelBehavior.always,
                hintText: serverFocusNode.hasFocus
                    ? 'UDP broadcast auto-discovery when empty'
                    : null,
                border: const OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
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
            const SizedBox(height: 24),
            const SectionHeader('Virtual Player Identity', color: Colors.greenAccent),
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
            const SizedBox(height: 16),
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
            const SizedBox(height: 24),
            const SectionHeader('Sync & Multiroom', color: Colors.greenAccent),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text('Auto-Sync to Active Player'),
              subtitle: const Text(
                'Automatically mirror audio streams from currently active LMS players',
              ),
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
              const SizedBox(height: 16),
              TextField(
                controller: ignoredPlayersCtrl,
                focusNode: ignoredPlayersFocusNode,
                decoration: const InputDecoration(
                  labelText: 'Ignored Players (comma separated)',
                  floatingLabelBehavior: FloatingLabelBehavior.always,
                  hintText: 'e.g. Living Room, Bathroom',
                  border: OutlineInputBorder(),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
