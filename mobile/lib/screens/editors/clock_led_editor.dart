import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../widgets/color_picker_tile.dart';
import '../../widgets/led_selectors.dart';
import '../../utils.dart';

class ClockLEDEditor extends StatefulWidget {
  const ClockLEDEditor({super.key});

  @override
  State<ClockLEDEditor> createState() => _ClockLEDEditorState();
}

class _ClockLEDEditorState extends State<ClockLEDEditor> {
  late int startLedHour;
  late int endLedHour;
  late int startLedMinute;
  late int endLedMinute;
  late Color ledHourColor;
  late Color ledMinuteColor;
  int ledsTotal = 100;

  bool _initialized = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_initialized) {
      final config = context.read<ConfigProvider>().config;
      if (config != null) {
        ledsTotal = config.ledsTotal;
        final c = config.clockLED;
        startLedHour = c.startLedHour;
        endLedHour = c.endLedHour;
        startLedMinute = c.startLedMinute;
        endLedMinute = c.endLedMinute;
        ledHourColor = fromRgbList(c.ledHour);
        ledMinuteColor = fromRgbList(c.ledMinute);
        _initialized = true;
      }
    }
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final config = provider.config;
    if (config == null) return;

    config.clockLED.startLedHour = startLedHour;
    config.clockLED.endLedHour = endLedHour;
    config.clockLED.startLedMinute = startLedMinute;
    config.clockLED.endLedMinute = endLedMinute;
    config.clockLED.ledHour = toRgbList(ledHourColor);
    config.clockLED.ledMinute = toRgbList(ledMinuteColor);

    provider.updateConfig(config).then((_) {
      if (mounted) Navigator.pop(context);
    });
  }

  @override
  Widget build(BuildContext context) {
    if (!_initialized) return const Scaffold(body: Center(child: CircularProgressIndicator()));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Clock Config'),
        actions: [IconButton(icon: const Icon(Icons.save), onPressed: _save)],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSectionHeader('Hour Hands'),
          LedRangeSelector(
            label: 'Hour Range',
            start: startLedHour,
            end: endLedHour,
            totalLeds: ledsTotal,
            onChanged: (s, e) => setState(() {
              startLedHour = s;
              endLedHour = e;
            }),
          ),
          const SizedBox(height: 16),
          ColorPickerTile(
            label: 'Hour Color',
            color: ledHourColor,
            onColorChanged: (c) => setState(() => ledHourColor = c),
          ),
          
          const SizedBox(height: 24),
          _buildSectionHeader('Minute Hands'),
          LedRangeSelector(
            label: 'Minute Range',
            start: startLedMinute,
            end: endLedMinute,
            totalLeds: ledsTotal,
            onChanged: (s, e) => setState(() {
              startLedMinute = s;
              endLedMinute = e;
            }),
          ),
          const SizedBox(height: 16),
          ColorPickerTile(
            label: 'Minute Color',
            color: ledMinuteColor,
            onColorChanged: (c) => setState(() => ledMinuteColor = c),
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
          color: Colors.blueAccent,
          fontWeight: FontWeight.bold,
          letterSpacing: 1.2,
        ),
      ),
    );
  }
}