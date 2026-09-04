import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../widgets/color_picker_tile.dart';
import '../../widgets/config_slider.dart';
import '../../widgets/section_header.dart';
import '../../models.dart';
import '../../utils.dart';

class CylonLEDEditor extends StatefulWidget {
  final CylonLEDConfig initialConfig;
  final int totalLeds;

  const CylonLEDEditor({
    super.key,
    required this.initialConfig,
    required this.totalLeds,
  });

  @override
  State<CylonLEDEditor> createState() => _CylonLEDEditorState();
}

class _CylonLEDEditorState extends State<CylonLEDEditor> {
  late int durationSec;
  late int delayMs;
  late double step;
  late int width;
  late Color eyeColor;

  @override
  void initState() {
    super.initState();
    final c = widget.initialConfig;
    durationSec = c.durationSec;
    delayMs = c.delayMs;
    step = c.step;
    width = c.width;
    eyeColor = fromRgbList(c.ledRGB);
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final currentFullConfig = provider.config;
    if (currentFullConfig == null) return;

    final updatedCylonConfig = currentFullConfig.cylonLED.copyWith(
      durationSec: durationSec,
      delayMs: delayMs,
      step: step,
      width: width,
      ledRGB: toRgbList(eyeColor),
    );

    provider
        .updateConfig(currentFullConfig.copyWith(cylonLED: updatedCylonConfig))
        .then((_) {
          if (mounted) Navigator.pop(context);
        });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Cylon Eye Config'),
        actions: [IconButton(icon: const Icon(Icons.save), onPressed: _save)],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const SectionHeader('Animation Settings', color: Colors.redAccent),
          ConfigSlider(
            label: 'Duration',
            value: durationSec.toDouble(),
            min: 0,
            max: 300,
            unit: 's',
            onChanged: (v) => setState(() => durationSec = v.toInt()),
            activeColor: Colors.redAccent,
          ),
          ConfigSlider(
            label: 'Speed (Delay)',
            value: delayMs.toDouble(),
            min: 0,
            max: 200,
            unit: 'ms',
            onChanged: (v) => setState(() => delayMs = v.toInt()),
            activeColor: Colors.redAccent,
          ),
          const SizedBox(height: 16),
          const SectionHeader('Appearance', color: Colors.redAccent),
          ConfigSlider(
            label: 'Eye Width',
            value: width.toDouble(),
            min: 1,
            max: (widget.totalLeds / 2).floorToDouble(),
            unit: 'px',
            onChanged: (v) => setState(() => width = v.toInt()),
            activeColor: Colors.redAccent,
          ),
          ConfigSlider(
            label: 'Step Size',
            value: step,
            min: 0.1,
            max: 5.0,
            isInt: false,
            onChanged: (v) =>
                setState(() => step = double.parse(v.toStringAsFixed(1))),
            activeColor: Colors.redAccent,
          ),
          ColorPickerTile(
            label: 'Eye Color',
            color: eyeColor,
            onColorChanged: (c) => setState(() => eyeColor = c),
          ),
        ],
      ),
    );
  }
}
