import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../widgets/color_picker_tile.dart';
import '../../utils.dart';

class SensorLEDEditor extends StatefulWidget {
  const SensorLEDEditor({super.key});

  @override
  State<SensorLEDEditor> createState() => _SensorLEDEditorState();
}

class _SensorLEDEditorState extends State<SensorLEDEditor> {
  // We maintain local state for editing, then commit on Save.
  // Ideally, we'd deep copy the config object.
  // For simplicity here, we'll read values into local variables.
  
  late int runUpDelayMs;
  late int runDownDelayMs;
  late int holdTimeSec;
  late Color ledColor;
  
  late bool latchEnabled;
  late int latchTriggerValue;
  late int latchTriggerDelaySec;
  late int latchTimeSec;
  late Color latchColor;

  bool _initialized = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_initialized) {
      final config = context.read<ConfigProvider>().config;
      if (config != null) {
        final s = config.sensorLED;
        runUpDelayMs = s.runUpDelayMs;
        runDownDelayMs = s.runDownDelayMs;
        holdTimeSec = s.holdTimeSec;
        ledColor = fromRgbList(s.ledRGB);
        
        latchEnabled = s.latchEnabled;
        latchTriggerValue = s.latchTriggerValue;
        latchTriggerDelaySec = s.latchTriggerDelaySec;
        latchTimeSec = s.latchTimeSec;
        latchColor = fromRgbList(s.latchLedRGB);
        
        _initialized = true;
      }
    }
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final config = provider.config;
    if (config == null) return;

    // Update the config object directly (in a real app, use deep copy/immutable)
    config.sensorLED.runUpDelayMs = runUpDelayMs;
    config.sensorLED.runDownDelayMs = runDownDelayMs;
    config.sensorLED.holdTimeSec = holdTimeSec;
    config.sensorLED.ledRGB = toRgbList(ledColor);
    
    config.sensorLED.latchEnabled = latchEnabled;
    config.sensorLED.latchTriggerValue = latchTriggerValue;
    config.sensorLED.latchTriggerDelaySec = latchTriggerDelaySec;
    config.sensorLED.latchTimeSec = latchTimeSec;
    config.sensorLED.latchLedRGB = toRgbList(latchColor);

    provider.updateConfig(config).then((_) {
      if (mounted) Navigator.pop(context);
    });
  }

  @override
  Widget build(BuildContext context) {
    if (!_initialized) return const Scaffold(body: Center(child: CircularProgressIndicator()));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Sensor LED Config'),
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
          _buildSectionHeader('Timing & Color'),
          _buildSlider('Run Up Delay', runUpDelayMs.toDouble(), 0, 1000, 'ms', (v) => setState(() => runUpDelayMs = v.toInt())),
          _buildSlider('Run Down Delay', runDownDelayMs.toDouble(), 0, 1000, 'ms', (v) => setState(() => runDownDelayMs = v.toInt())),
          _buildSlider('Hold Time', holdTimeSec.toDouble(), 0, 60, 's', (v) => setState(() => holdTimeSec = v.toInt())),
          ColorPickerTile(
            label: 'Active Color',
            color: ledColor,
            onColorChanged: (c) => setState(() => ledColor = c),
          ),
          
          const SizedBox(height: 24),
          _buildSectionHeader('Latch Mode'),
          SwitchListTile(
            title: const Text('Enable Latch'),
            value: latchEnabled,
            onChanged: (v) => setState(() => latchEnabled = v),
          ),
          if (latchEnabled) ...[
             _buildSlider('Trigger Value', latchTriggerValue.toDouble(), 0, 1023, '', (v) => setState(() => latchTriggerValue = v.toInt())),
             _buildSlider('Trigger Delay', latchTriggerDelaySec.toDouble(), 0, 10, 's', (v) => setState(() => latchTriggerDelaySec = v.toInt())),
             _buildSlider('Latch Duration', latchTimeSec.toDouble(), 0, 600, 's', (v) => setState(() => latchTimeSec = v.toInt())),
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

  Widget _buildSectionHeader(String title) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Text(
        title.toUpperCase(),
        style: const TextStyle(
          color: Colors.deepPurpleAccent,
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
            Text('${value.toInt()}$unit', style: const TextStyle(fontWeight: FontWeight.bold)),
          ],
        ),
        Slider(
          value: value,
          min: min,
          max: max,
          divisions: (max - min).toInt(),
          onChanged: onChanged,
          activeColor: Colors.deepPurpleAccent,
        ),
      ],
    );
  }
}
