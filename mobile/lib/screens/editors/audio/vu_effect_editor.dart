import 'package:flutter/material.dart';
import '../../../models.dart';
import '../../../utils.dart';
import '../../../widgets/color_picker_tile.dart';
import '../../../widgets/config_slider.dart';
import '../../../widgets/section_header.dart';

class AudioVUEffectEditor extends StatefulWidget {
  final AudioVUConfig initialConfig;

  const AudioVUEffectEditor({
    super.key,
    required this.initialConfig,
  });

  @override
  State<AudioVUEffectEditor> createState() => _AudioVUEffectEditorState();
}

class _AudioVUEffectEditorState extends State<AudioVUEffectEditor> {
  late Color ledLow;
  late Color ledMid;
  late Color ledHigh;
  late int switchStep1;
  late int switchStep2;
  late bool peakHoldEnabled;
  late int peakHoldTimeMs;
  late double peakDecayRate;

  @override
  void initState() {
    super.initState();
    final vu = widget.initialConfig;
    ledLow = fromRgbList(vu.ledLow);
    ledMid = fromRgbList(vu.ledMid);
    ledHigh = fromRgbList(vu.ledHigh);

    switchStep1 = vu.switchSteps.isNotEmpty ? vu.switchSteps[0] : 60;
    switchStep2 = vu.switchSteps.length > 1 ? vu.switchSteps[1] : 80;
    peakHoldEnabled = vu.peakHoldEnabled;
    peakHoldTimeMs = vu.peakHoldTimeMs > 0 ? vu.peakHoldTimeMs : 250;
    peakDecayRate = vu.peakDecayRate > 0 ? vu.peakDecayRate : 15.0;
  }

  void _done() {
    final updated = AudioVUConfig(
      ledLow: toRgbList(ledLow),
      ledMid: toRgbList(ledMid),
      ledHigh: toRgbList(ledHigh),
      switchSteps: [switchStep1, switchStep2],
      peakHoldEnabled: peakHoldEnabled,
      peakHoldTimeMs: peakHoldTimeMs,
      peakDecayRate: peakDecayRate,
    );
    Navigator.pop(context, updated);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('VU Meter Effect'),
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
            const SectionHeader('Gradient Colors', color: Colors.greenAccent),
            ColorPickerTile(
              label: 'Low Color (0%..Step 1)',
              color: ledLow,
              onColorChanged: (c) => setState(() => ledLow = c),
            ),
            ColorPickerTile(
              label: 'Mid Color (Step 1..Step 2)',
              color: ledMid,
              onColorChanged: (c) => setState(() => ledMid = c),
            ),
            ColorPickerTile(
              label: 'High Color (Step 2..100%)',
              color: ledHigh,
              onColorChanged: (c) => setState(() => ledHigh = c),
            ),
            const SizedBox(height: 24),
            const SectionHeader('Threshold Steps', color: Colors.greenAccent),
            ConfigSlider(
              label: 'Mid Threshold Step',
              value: switchStep1.toDouble(),
              min: 10,
              max: (switchStep2 - 5).clamp(10, 90).toDouble(),
              unit: '%',
              onChanged: (v) => setState(() => switchStep1 = v.toInt()),
              activeColor: Colors.greenAccent,
            ),
            const SizedBox(height: 8),
            ConfigSlider(
              label: 'High Threshold Step',
              value: switchStep2.toDouble(),
              min: (switchStep1 + 5).clamp(15, 95).toDouble(),
              max: 95,
              unit: '%',
              onChanged: (v) => setState(() => switchStep2 = v.toInt()),
              activeColor: Colors.greenAccent,
            ),
            const SizedBox(height: 24),
            const SectionHeader('Peak Hold & Decay', color: Colors.greenAccent),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text('Peak Hold Indicator'),
              subtitle: const Text('Keeps a peak LED marker suspended before decaying'),
              value: peakHoldEnabled,
              onChanged: (val) => setState(() => peakHoldEnabled = val),
              activeThumbColor: Colors.greenAccent,
            ),
            if (peakHoldEnabled) ...[
              const SizedBox(height: 8),
              ConfigSlider(
                label: 'Peak Hold Time',
                value: peakHoldTimeMs.toDouble(),
                min: 50,
                max: 1000,
                unit: 'ms',
                onChanged: (v) => setState(() => peakHoldTimeMs = v.toInt()),
                activeColor: Colors.greenAccent,
              ),
              const SizedBox(height: 8),
              ConfigSlider(
                label: 'Peak Decay Rate',
                value: peakDecayRate,
                min: 1,
                max: 50,
                unit: 'LEDs/s',
                onChanged: (v) => setState(() => peakDecayRate = v),
                activeColor: Colors.greenAccent,
              ),
            ],
          ],
        ),
      ),
    );
  }
}
