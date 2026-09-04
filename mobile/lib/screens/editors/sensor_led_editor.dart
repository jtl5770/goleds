import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../widgets/color_picker_tile.dart';
import '../../widgets/config_slider.dart';
import '../../widgets/section_header.dart';
import '../../models.dart';
import '../../utils.dart';

class SensorLEDEditor extends StatefulWidget {
  final SensorLEDConfig initialConfig;

  const SensorLEDEditor({super.key, required this.initialConfig});

  @override
  State<SensorLEDEditor> createState() => _SensorLEDEditorState();
}

class _SensorLEDEditorState extends State<SensorLEDEditor> {
  late int runUpDelayMs;
  late int runDownDelayMs;
  late int holdTimeSec;
  late Color ledColor;

  late bool latchEnabled;
  late int latchTriggerValue;
  late int latchTriggerDelaySec;
  late int latchTimeSec;
  late Color latchColor;

  @override
  void initState() {
    super.initState();
    final s = widget.initialConfig;
    runUpDelayMs = s.runUpDelayMs;
    runDownDelayMs = s.runDownDelayMs;
    holdTimeSec = s.holdTimeSec;
    ledColor = fromRgbList(s.ledRGB);

    latchEnabled = s.latchEnabled;
    latchTriggerValue = s.latchTriggerValue;
    latchTriggerDelaySec = s.latchTriggerDelaySec;
    latchTimeSec = s.latchTimeSec;
    latchColor = fromRgbList(s.latchLedRGB);
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final currentFullConfig = provider.config;
    if (currentFullConfig == null) return;

    final updatedSensorConfig = currentFullConfig.sensorLED.copyWith(
      runUpDelayMs: runUpDelayMs,
      runDownDelayMs: runDownDelayMs,
      holdTimeSec: holdTimeSec,
      ledRGB: toRgbList(ledColor),
      latchEnabled: latchEnabled,
      latchTriggerValue: latchTriggerValue,
      latchTriggerDelaySec: latchTriggerDelaySec,
      latchTimeSec: latchTimeSec,
      latchLedRGB: toRgbList(latchColor),
    );

    provider
        .updateConfig(
          currentFullConfig.copyWith(sensorLED: updatedSensorConfig),
        )
        .then((_) {
          if (mounted) Navigator.pop(context);
        });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Sensor LED Config'),
        actions: [IconButton(icon: const Icon(Icons.save), onPressed: _save)],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const SectionHeader('Timing & Color', color: Colors.deepPurpleAccent),
          ConfigSlider(
            label: 'Run Up Delay',
            value: runUpDelayMs.toDouble(),
            min: 0,
            max: 50,
            unit: 'ms',
            onChanged: (v) => setState(() => runUpDelayMs = v.toInt()),
          ),
          ConfigSlider(
            label: 'Run Down Delay',
            value: runDownDelayMs.toDouble(),
            min: 0,
            max: 50,
            unit: 'ms',
            onChanged: (v) => setState(() => runDownDelayMs = v.toInt()),
          ),
          ConfigSlider(
            label: 'Hold Time',
            value: holdTimeSec.toDouble(),
            min: 0,
            max: 60,
            unit: 's',
            onChanged: (v) => setState(() => holdTimeSec = v.toInt()),
          ),
          ColorPickerTile(
            label: 'Active Color',
            color: ledColor,
            onColorChanged: (c) => setState(() => ledColor = c),
          ),
          const SizedBox(height: 24),
          const SectionHeader('Latch Mode', color: Colors.deepPurpleAccent),
          SwitchListTile(
            title: const Text('Enable Latch'),
            value: latchEnabled,
            onChanged: (v) => setState(() => latchEnabled = v),
          ),
          if (latchEnabled) ...[
            ConfigSlider(
              label: 'Trigger Value',
              value: latchTriggerValue.toDouble(),
              min: 0,
              max: 1023,
              onChanged: (v) => setState(() => latchTriggerValue = v.toInt()),
            ),
            ConfigSlider(
              label: 'Trigger Delay',
              value: latchTriggerDelaySec.toDouble(),
              min: 0,
              max: 10,
              unit: 's',
              onChanged: (v) =>
                  setState(() => latchTriggerDelaySec = v.toInt()),
            ),
            ConfigSlider(
              label: 'Latch Duration',
              value: latchTimeSec.toDouble(),
              min: 0,
              max: 3600,
              unit: 's',
              onChanged: (v) => setState(() => latchTimeSec = v.toInt()),
            ),
            ColorPickerTile(
              label: 'Latch Color',
              color: latchColor,
              onColorChanged: (c) => setState(() => latchColor = c),
            ),
          ],
        ],
      ),
    );
  }
}
