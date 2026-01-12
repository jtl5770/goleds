import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../widgets/color_picker_tile.dart';
import '../../utils.dart';

class CylonLEDEditor extends StatefulWidget {
  const CylonLEDEditor({super.key});

  @override
  State<CylonLEDEditor> createState() => _CylonLEDEditorState();
}

class _CylonLEDEditorState extends State<CylonLEDEditor> {
  late int durationSec;
  late int delayMs;
  late double step;
  late int width;
  late Color eyeColor;
  int ledsTotal = 100;

  bool _initialized = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_initialized) {
      final config = context.read<ConfigProvider>().config;
      if (config != null) {
        ledsTotal = config.ledsTotal;
        final c = config.cylonLED;
        durationSec = c.durationSec;
        delayMs = c.delayMs;
        step = c.step;
        width = c.width;
        eyeColor = fromRgbList(c.ledRGB);
        _initialized = true;
      }
    }
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final config = provider.config;
    if (config == null) return;

    config.cylonLED.durationSec = durationSec;
    config.cylonLED.delayMs = delayMs;
    config.cylonLED.step = step;
    config.cylonLED.width = width;
    config.cylonLED.ledRGB = toRgbList(eyeColor);

    provider.updateConfig(config).then((_) {
      if (mounted) Navigator.pop(context);
    });
  }

  @override
  Widget build(BuildContext context) {
    if (!_initialized) return const Scaffold(body: Center(child: CircularProgressIndicator()));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Cylon Eye Config'),
        actions: [
          IconButton(
            icon: const Icon(Icons.save),
            onPressed: _save,
          )
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSectionHeader('Animation Settings'),
          _buildSlider('Duration', durationSec.toDouble(), 0, 300, 's', (v) => setState(() => durationSec = v.toInt())),
          _buildSlider('Speed (Delay)', delayMs.toDouble(), 0, 200, 'ms', (v) => setState(() => delayMs = v.toInt())),
          
          const SizedBox(height: 16),
          _buildSectionHeader('Appearance'),
          _buildSlider('Eye Width', width.toDouble(), 1, (ledsTotal / 2).floorToDouble(), 'px', (v) => setState(() => width = v.toInt())),
          // Step is a double, maybe use a slider with divisions
          _buildSlider('Step Size', step, 0.1, 5.0, '', (v) => setState(() => step = double.parse(v.toStringAsFixed(1)))),
          
          ColorPickerTile(
            label: 'Eye Color',
            color: eyeColor,
            onColorChanged: (c) => setState(() => eyeColor = c),
          ),
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
          color: Colors.redAccent,
          fontWeight: FontWeight.bold,
          letterSpacing: 1.2,
        ),
      ),
    );
  }

  Widget _buildSlider(String label, double value, double min, double max, String unit, ValueChanged<double> onChanged) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label),
            Text('${value.toStringAsFixed(value is int ? 0 : 1)}$unit', style: const TextStyle(fontWeight: FontWeight.bold)),
          ],
        ),
        Slider(
          value: value,
          min: min,
          max: max,
          divisions: (max - min) ~/ (unit.isEmpty ? 0.1 : 1),
          onChanged: onChanged,
          activeColor: Colors.redAccent,
        ),
      ],
    );
  }
}
