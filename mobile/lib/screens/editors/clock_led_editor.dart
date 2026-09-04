import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../widgets/color_picker_tile.dart';
import '../../widgets/led_selectors.dart';
import '../../widgets/section_header.dart';
import '../../models.dart';
import '../../utils.dart';

class ClockLEDEditor extends StatefulWidget {
  final ClockLEDConfig initialConfig;
  final int totalLeds;

  const ClockLEDEditor({
    super.key,
    required this.initialConfig,
    required this.totalLeds,
  });

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

  @override
  void initState() {
    super.initState();
    final c = widget.initialConfig;
    startLedHour = c.startLedHour;
    endLedHour = c.endLedHour;
    startLedMinute = c.startLedMinute;
    endLedMinute = c.endLedMinute;
    ledHourColor = fromRgbList(c.ledHour);
    ledMinuteColor = fromRgbList(c.ledMinute);
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final currentFullConfig = provider.config;
    if (currentFullConfig == null) return;

    final updatedClockConfig = currentFullConfig.clockLED.copyWith(
      startLedHour: startLedHour,
      endLedHour: endLedHour,
      startLedMinute: startLedMinute,
      endLedMinute: endLedMinute,
      ledHour: toRgbList(ledHourColor),
      ledMinute: toRgbList(ledMinuteColor),
    );

    provider
        .updateConfig(currentFullConfig.copyWith(clockLED: updatedClockConfig))
        .then((_) {
          if (mounted) Navigator.pop(context);
        });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Clock Config'),
        actions: [IconButton(icon: const Icon(Icons.save), onPressed: _save)],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const SectionHeader('Hour Hands', color: Colors.blueAccent),
          LedRangeSelector(
            label: 'Hour Range',
            start: startLedHour,
            end: endLedHour,
            totalLeds: widget.totalLeds,
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
          const SectionHeader('Minute Hands', color: Colors.blueAccent),
          LedRangeSelector(
            label: 'Minute Range',
            start: startLedMinute,
            end: endLedMinute,
            totalLeds: widget.totalLeds,
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
}
