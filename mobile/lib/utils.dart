import 'package:flutter/material.dart';

Color fromRgbList(List<double> rgb) {
  if (rgb.length < 3) return Colors.black;
  return Color.fromARGB(255, rgb[0].toInt(), rgb[1].toInt(), rgb[2].toInt());
}

List<double> toRgbList(Color c) {
  // Use .r, .g, .b which are 0.0-1.0 in newer Flutter versions
  return [(c.r * 255), (c.g * 255), (c.b * 255)];
}