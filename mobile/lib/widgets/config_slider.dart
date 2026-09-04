import 'package:flutter/material.dart';

class ConfigSlider extends StatelessWidget {
  final String label;
  final double value;
  final double min;
  final double max;
  final String unit;
  final ValueChanged<double> onChanged;
  final Color activeColor;
  final bool isInt;

  const ConfigSlider({
    super.key,
    required this.label,
    required this.value,
    required this.min,
    required this.max,
    required this.onChanged,
    this.unit = '',
    this.activeColor = Colors.deepPurpleAccent,
    this.isInt = true,
  });

  void _showEditDialog(BuildContext context) {
    final controller = TextEditingController(
      text: isInt ? value.toInt().toString() : value.toStringAsFixed(1),
    );

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Edit $label', style: const TextStyle(fontSize: 16)),
        content: SizedBox(
          width: 220,
          child: TextField(
            controller: controller,
            autofocus: true,
            keyboardType: TextInputType.numberWithOptions(
              decimal: !isInt,
              signed: min < 0,
            ),
            decoration: InputDecoration(
              labelText: 'Value',
              floatingLabelBehavior: FloatingLabelBehavior.always,
              suffixText: unit.isNotEmpty ? unit : null,
              border: const OutlineInputBorder(),
            ),
            onSubmitted: (text) => _apply(ctx, text),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () => _apply(ctx, controller.text),
            child: const Text('OK'),
          ),
        ],
      ),
    );
  }

  void _apply(BuildContext ctx, String text) {
    final parsed = double.tryParse(text);
    if (parsed != null) {
      final clamped = parsed.clamp(min, max);
      onChanged(isInt ? clamped.roundToDouble() : clamped);
    }
    Navigator.pop(ctx);
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label),
            InkWell(
              onTap: () => _showEditDialog(context),
              borderRadius: BorderRadius.circular(4),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
                child: Text(
                  '${isInt ? value.toInt() : value.toStringAsFixed(1)}$unit',
                  style: const TextStyle(
                    fontWeight: FontWeight.bold,
                    decoration: TextDecoration.underline,
                    decorationStyle: TextDecorationStyle.dotted,
                  ),
                ),
              ),
            ),
          ],
        ),
        Slider(
          value: value.clamp(min, max),
          min: min,
          max: max,
          divisions: isInt ? (max - min).toInt() : (max - min) ~/ 0.1,
          onChanged: onChanged,
          activeColor: activeColor,
        ),
      ],
    );
  }
}
