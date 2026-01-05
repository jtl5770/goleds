import 'package:flutter/material.dart';
import 'rgb_input_picker.dart';
import 'led_preview.dart';

class ColorPickerTile extends StatelessWidget {
  final String label;
  final Color color;
  final ValueChanged<Color> onColorChanged;

  const ColorPickerTile({
    super.key,
    required this.label,
    required this.color,
    required this.onColorChanged,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      title: Text(label),
      trailing: LedPreview(color: color, size: 36),
      onTap: () {
        showDialog(
          context: context,
          builder: (ctx) => AlertDialog(
            title: Text('Pick $label'),
            content: RgbInputPicker(
              initialColor: color,
              onColorChanged: onColorChanged,
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(ctx).pop(),
                child: const Text('Done'),
              ),
            ],
          ),
        );
      },
    );
  }
}
