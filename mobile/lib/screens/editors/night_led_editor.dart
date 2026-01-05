import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../utils.dart';
import '../../widgets/rgb_input_picker.dart';
import '../../widgets/led_preview.dart';

class NightLEDEditor extends StatefulWidget {
  const NightLEDEditor({super.key});

  @override
  State<NightLEDEditor> createState() => _NightLEDEditorState();
}

class _NightLEDEditorState extends State<NightLEDEditor> {
  late double latitude;
  late double longitude;
  late List<Color> colors;

  bool _initialized = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_initialized) {
      final config = context.read<ConfigProvider>().config;
      if (config != null) {
        final n = config.nightLED;
        latitude = n.latitude;
        longitude = n.longitude;
        colors = n.ledRGB.map((rgb) => fromRgbList(rgb)).toList();
        _initialized = true;
      }
    }
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final config = provider.config;
    if (config == null) return;

    config.nightLED.latitude = latitude;
    config.nightLED.longitude = longitude;
    config.nightLED.ledRGB = colors.map((c) => toRgbList(c)).toList();

    provider.updateConfig(config).then((_) {
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
             child: const Text('Delete', style: TextStyle(color: Colors.red)),
          ),
          TextButton(
            onPressed: () {
              setState(() {
                colors[index] = tempColor;
              });
              Navigator.pop(ctx);
            },
            child: const Text('Save'),
          ),
        ],
      ),
    );
  }

  void _addColor() {
    setState(() {
      colors.add(Colors.deepPurple); // Default new color
    });
  }

  @override
  Widget build(BuildContext context) {
    if (!_initialized) return const Scaffold(body: Center(child: CircularProgressIndicator()));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Night Light Config'),
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
          _buildSectionHeader('Location'),
          const Text('Coordinates are used to calculate sunset/sunrise times.', style: TextStyle(color: Colors.grey)),
          const SizedBox(height: 16),
          _buildNumberInput('Latitude', latitude, (v) => latitude = v),
          const SizedBox(height: 8),
          _buildNumberInput('Longitude', longitude, (v) => longitude = v),
          
          const SizedBox(height: 32),
          _buildSectionHeader('Night Sequence'),
          const Text('Colors cycle evenly throughout the night duration.', style: TextStyle(color: Colors.grey)),
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
                    border: Border.all(color: Colors.grey, width: 2, style: BorderStyle.solid),
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

  Widget _buildSectionHeader(String title) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Text(
        title.toUpperCase(),
        style: const TextStyle(
          color: Colors.orangeAccent,
          fontWeight: FontWeight.bold,
          letterSpacing: 1.2,
        ),
      ),
    );
  }

  Widget _buildNumberInput(String label, double val, ValueChanged<double> onChanged) {
    return TextFormField(
      initialValue: val.toString(),
      keyboardType: const TextInputType.numberWithOptions(decimal: true, signed: true),
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
