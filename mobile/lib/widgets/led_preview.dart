import 'package:flutter/material.dart';
import '../utils.dart';

class LedPreview extends StatelessWidget {
  final Color color;
  final double size;
  final bool hasBorder;

  const LedPreview({
    super.key,
    required this.color,
    this.size = 40,
    this.hasBorder = true,
  });

  @override
  Widget build(BuildContext context) {
    final displayColor = toDisplayColor(color);
    final shadows = getGlowShadow(color, scale: size / 40);

    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: displayColor,
        shape: BoxShape.circle,
        border: hasBorder
            ? Border.all(color: Colors.white, width: size * 0.05)
            : null,
        boxShadow: shadows,
      ),
    );
  }
}
