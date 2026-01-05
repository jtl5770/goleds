import 'package:flutter/material.dart';
import 'package:flutter_colorpicker/flutter_colorpicker.dart';

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
      trailing: Container(
        width: 40,
        height: 40,
        decoration: BoxDecoration(
          color: color,
          shape: BoxShape.circle,
          border: Border.all(color: Colors.white24),
        ),
      ),
      onTap: () {
        showDialog(
          context: context,
          builder: (ctx) => AlertDialog(
            title: Text('Pick $label'),
            content: SingleChildScrollView(
              child: ColorPicker(
                pickerColor: color,
                onColorChanged: onColorChanged,
                pickerAreaHeightPercent: 0.7,
                enableAlpha: false,
                labelTypes: const [ColorLabelType.rgb],
                displayThumbColor: true,
                hexInputBar: false,
              ),
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
