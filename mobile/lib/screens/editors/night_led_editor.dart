import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../utils.dart';
import '../../models.dart';
import '../../widgets/rgb_input_picker.dart';
import '../../widgets/led_preview.dart';
import '../../widgets/section_header.dart';

class NightLEDEditor extends StatefulWidget {
  final NightLEDConfig initialConfig;

  const NightLEDEditor({super.key, required this.initialConfig});

  @override
  State<NightLEDEditor> createState() => _NightLEDEditorState();
}

class _NightLEDEditorState extends State<NightLEDEditor> {
  late double latitude;
  late double longitude;
  late List<Color> colors;

  @override
  void initState() {
    super.initState();
    final n = widget.initialConfig;
    latitude = n.latitude;
    longitude = n.longitude;
    colors = n.ledRGB.map((rgb) => fromRgbList(rgb)).toList();
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final currentFullConfig = provider.config;
    if (currentFullConfig == null) return;

    final updatedNightConfig = currentFullConfig.nightLED.copyWith(
      latitude: latitude,
      longitude: longitude,
      ledRGB: colors.map((c) => toRgbList(c)).toList(),
    );

    provider
        .updateConfig(currentFullConfig.copyWith(nightLED: updatedNightConfig))
        .then((_) {
          if (mounted) Navigator.pop(context);
        });
  }

  void _editColor(int index) {
    Color tempColor = colors[index];
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Pick Night Color'),
        content: RgbInputPicker(
          initialColor: tempColor,
          onColorChanged: (c) => tempColor = c,
        ),
        actions: [
          TextButton(
            onPressed: () {
              setState(() {
                colors.removeAt(index);
              });
              Navigator.pop(ctx);
            },
            child: const Text('DELETE', style: TextStyle(color: Colors.red)),
          ),
          ElevatedButton(
            onPressed: () {
              setState(() {
                colors[index] = tempColor;
              });
              Navigator.pop(ctx);
            },
            child: const Text('DONE'),
          ),
        ],
      ),
    );
  }

  void _addColor() {
    setState(() {
      colors.add(Colors.deepPurple);
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Night Light Config'),
        actions: [IconButton(icon: const Icon(Icons.save), onPressed: _save)],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const SectionHeader(
            'Location',
            color: Colors.orangeAccent,
            padding: EdgeInsets.only(bottom: 8),
          ),
          const Text(
            'Coordinates are used to calculate sunset/sunrise times.',
            style: TextStyle(color: Colors.grey),
          ),
          const SizedBox(height: 16),
          _buildNumberInput('Latitude', latitude, (v) => latitude = v),
          const SizedBox(height: 8),
          _buildNumberInput('Longitude', longitude, (v) => longitude = v),
          const SizedBox(height: 32),
          const SectionHeader(
            'Night Sequence',
            color: Colors.orangeAccent,
            padding: EdgeInsets.only(bottom: 8),
          ),
          const Text(
            'Colors cycle evenly throughout the night duration.',
            style: TextStyle(color: Colors.grey),
          ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 16,
            runSpacing: 16,
            children: [
              ...List.generate(colors.length, (index) {
                return GestureDetector(
                  onTap: () => _editColor(index),
                  child: LedPreview(color: colors[index], size: 60),
                );
              }),
              GestureDetector(
                onTap: _addColor,
                child: Container(
                  width: 60,
                  height: 60,
                  decoration: BoxDecoration(
                    color: Colors.transparent,
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: Colors.grey,
                      width: 2,
                      style: BorderStyle.solid,
                    ),
                  ),
                  child: const Icon(Icons.add, color: Colors.grey),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildNumberInput(
    String label,
    double val,
    ValueChanged<double> onChanged,
  ) {
    return TextFormField(
      initialValue: val.toString(),
      keyboardType: const TextInputType.numberWithOptions(
        decimal: true,
        signed: true,
      ),
      decoration: InputDecoration(
        labelText: label,
        border: const OutlineInputBorder(),
        suffixIcon: const Icon(Icons.map, color: Colors.orangeAccent),
      ),
      onChanged: (v) {
        final d = double.tryParse(v);
        if (d != null) onChanged(d);
      },
    );
  }
}
