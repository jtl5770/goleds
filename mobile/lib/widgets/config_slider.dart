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

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label),
            Text(
              '${isInt ? value.toInt() : value.toStringAsFixed(1)}$unit',
              style: const TextStyle(fontWeight: FontWeight.bold),
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
