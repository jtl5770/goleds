import 'package:flutter/material.dart';
import 'dart:math' as math;

Color fromRgbList(List<double> rgb) {
  if (rgb.length < 3) return Colors.black;
  return Color.fromARGB(255, rgb[0].toInt(), rgb[1].toInt(), rgb[2].toInt());
}

List<double> toRgbList(Color c) {
  // Use .r, .g, .b which are 0.0-1.0 in newer Flutter versions
  return [(c.r * 255), (c.g * 255), (c.b * 255)];
}

/// Clamps the minimum brightness of a color so it's always visible on a black screen,
/// while preserving its hue and saturation.
Color toDisplayColor(Color actualColor) {
  final hsv = HSVColor.fromColor(actualColor);
  // Make the minimum hsv.value 0.4 and scale accordingly with lower slope.
  // But let's make sure black remains black.'
  if (hsv.value == 0) {
    return Color(0xFF000000);
  }
  return hsv.withValue(0.4 + (0.6 * hsv.value)).toColor();
}

/// Generates a glow effect proportional to the actual color's intensity.
List<BoxShadow> getGlowShadow(Color actualColor, {double scale = 1.0}) {
  final hsv = HSVColor.fromColor(actualColor);

  // Intensity is represented by the HSV Value (0.0 to 1.0)
  double intensity = hsv.value;

  // No intensity, no glow
  if (intensity <= 0) return [];

  // Use square root to make the glow ramp up much faster for low intensities
  double glowFactor = math.sqrt(intensity);

  return [
    // Main soft glow
    BoxShadow(
      color: actualColor.withValues(alpha: 0.4 + (glowFactor * 0.6)),
      blurRadius: (8 + (glowFactor * 32)) * scale,
      spreadRadius: (2 + (glowFactor * 8)) * scale,
    ),
    // Inner "bloom" for extra punch
    BoxShadow(
      color: Colors.white.withValues(alpha: glowFactor * 0.4),
      blurRadius: (4 + (glowFactor * 12)) * scale,
      spreadRadius: 0,
    ),
  ];
}
