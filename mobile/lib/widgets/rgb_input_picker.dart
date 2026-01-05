import 'package:flutter/material.dart';
import 'package:flutter_colorpicker/flutter_colorpicker.dart';

class RgbInputPicker extends StatefulWidget {
  final Color initialColor;
  final ValueChanged<Color> onColorChanged;

  const RgbInputPicker({
    super.key,
    required this.initialColor,
    required this.onColorChanged,
  });

  @override
  State<RgbInputPicker> createState() => _RgbInputPickerState();
}

class _RgbInputPickerState extends State<RgbInputPicker> {
  late Color currentColor;
  late TextEditingController rCtrl, gCtrl, bCtrl;

  @override
  void initState() {
    super.initState();
    currentColor = widget.initialColor;
    rCtrl = TextEditingController(text: (currentColor.r * 255).round().toString());
    gCtrl = TextEditingController(text: (currentColor.g * 255).round().toString());
    bCtrl = TextEditingController(text: (currentColor.b * 255).round().toString());
  }

  @override
  void dispose() {
    rCtrl.dispose();
    gCtrl.dispose();
    bCtrl.dispose();
    super.dispose();
  }

  void _updateFromPicker(Color color) {
    if (color == currentColor) return;
    setState(() {
      currentColor = color;
      // Use selection-aware update to avoid cursor jumping if focused
      _updateTextIfChanged(rCtrl, (color.r * 255).round().toString());
      _updateTextIfChanged(gCtrl, (color.g * 255).round().toString());
      _updateTextIfChanged(bCtrl, (color.b * 255).round().toString());
    });
    widget.onColorChanged(color);
  }

  void _updateTextIfChanged(TextEditingController ctrl, String newValue) {
    if (ctrl.text != newValue) {
      ctrl.text = newValue;
    }
  }

  void _updateFromText() {
    int r = int.tryParse(rCtrl.text) ?? 0;
    int g = int.tryParse(gCtrl.text) ?? 0;
    int b = int.tryParse(bCtrl.text) ?? 0;
    
    Color newColor = Color.fromARGB(255, r.clamp(0, 255), g.clamp(0, 255), b.clamp(0, 255));
    if (newColor.r != currentColor.r || newColor.g != currentColor.g || newColor.b != currentColor.b) {
      setState(() {
        currentColor = newColor;
      });
      widget.onColorChanged(newColor);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        ColorPicker(
          pickerColor: currentColor,
          onColorChanged: _updateFromPicker,
          enableAlpha: false,
          labelTypes: const [], // Hide the library's labels
          displayThumbColor: true,
          pickerAreaHeightPercent: 0.7,
        ),
        const SizedBox(height: 16),
        Row(
          children: [
            _buildField('R', rCtrl),
            _buildField('G', gCtrl),
            _buildField('B', bCtrl),
          ],
        ),
        const SizedBox(height: 8),
      ],
    );
  }

  Widget _buildField(String label, TextEditingController ctrl) {
    return Expanded(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4),
        child: TextFormField(
          controller: ctrl,
          keyboardType: TextInputType.number,
          textAlign: TextAlign.center,
          style: const TextStyle(fontFamily: 'monospace', fontWeight: FontWeight.bold),
          decoration: InputDecoration(
            labelText: label,
            border: const OutlineInputBorder(),
            contentPadding: const EdgeInsets.symmetric(vertical: 8),
          ),
          onChanged: (_) => _updateFromText(),
        ),
      ),
    );
  }
}
