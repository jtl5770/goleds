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
  late TextEditingController deviceCtrl;
  late int startLedLeft, endLedLeft;
  late int startLedRight, endLedRight;
  late Color ledGreen, ledYellow, ledRed;
  late int sampleRate;
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
        deviceCtrl = TextEditingController(text: a.device);
        startLedLeft = a.startLedLeft;
        endLedLeft = a.endLedLeft;
        startLedRight = a.startLedRight;
        endLedRight = a.endLedRight;
        ledGreen = fromRgbList(a.ledGreen);
        ledYellow = fromRgbList(a.ledYellow);
        ledRed = fromRgbList(a.ledRed);
        sampleRate = a.sampleRate;
        updateFreqMs = a.updateFreqMs;
        minDB = a.minDB;
        maxDB = a.maxDB;
        _initialized = true;
      }
    }
  }

  @override
  void dispose() {
    deviceCtrl.dispose();
    super.dispose();
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final config = provider.config;
    if (config == null) return;

    config.audioLED.device = deviceCtrl.text;
    config.audioLED.startLedLeft = startLedLeft;
    config.audioLED.endLedLeft = endLedLeft;
    config.audioLED.startLedRight = startLedRight;
    config.audioLED.endLedRight = endLedRight;
    config.audioLED.ledGreen = toRgbList(ledGreen);
    config.audioLED.ledYellow = toRgbList(ledYellow);
    config.audioLED.ledRed = toRgbList(ledRed);
    config.audioLED.sampleRate = sampleRate;
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
          _buildSectionHeader('Audio Source'),
          TextField(
            controller: deviceCtrl,
            decoration: const InputDecoration(
              labelText: 'ALSA/PulseAudio Device Name',
              border: OutlineInputBorder(),
              helperText: 'e.g., "pulse", "default", "hw:1,0"',
            ),
          ),
          const SizedBox(height: 16),
          ConfigSlider(
            label: 'Sample Rate',
            value: sampleRate.toDouble(),
            min: 8000,
            max: 48000,
            unit: 'Hz',
            onChanged: (v) => setState(() => sampleRate = v.toInt()),
            activeColor: Colors.greenAccent,
          ),
          ConfigSlider(
            label: 'Update Frequency',
            value: updateFreqMs.toDouble(),
            min: 10,
            max: 200,
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