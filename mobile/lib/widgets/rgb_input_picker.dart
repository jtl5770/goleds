import 'package:flutter/material.dart';
import 'package:flutter_colorpicker/flutter_colorpicker.dart';
import 'led_preview.dart';

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
  late HSVColor hsvColor;
  late TextEditingController rCtrl, gCtrl, bCtrl;

  @override
  void initState() {
    super.initState();
    hsvColor = HSVColor.fromColor(widget.initialColor);
    final color = widget.initialColor;
    rCtrl = TextEditingController(text: (color.r * 255).round().toString());
    gCtrl = TextEditingController(text: (color.g * 255).round().toString());
    bCtrl = TextEditingController(text: (color.b * 255).round().toString());
  }

  @override
  void dispose() {
    rCtrl.dispose();
    gCtrl.dispose();
    bCtrl.dispose();
    super.dispose();
  }

  void _updateColor(HSVColor newHsv) {
    final clampedHsv = newHsv.withHue(newHsv.hue.clamp(0.0, 360.0));
    if (clampedHsv == hsvColor) return;

    setState(() {
      hsvColor = clampedHsv;
      final color = clampedHsv.toColor();
      _updateTextIfChanged(rCtrl, (color.r * 255).round().toString());
      _updateTextIfChanged(gCtrl, (color.g * 255).round().toString());
      _updateTextIfChanged(bCtrl, (color.b * 255).round().toString());
    });
    widget.onColorChanged(clampedHsv.toColor());
  }

  void _updateTextIfChanged(TextEditingController ctrl, String newValue) {
    if (ctrl.text != newValue) {
      ctrl.text = newValue;
    }
  }

  void _updateFromText() {
    int r = (int.tryParse(rCtrl.text) ?? 0).clamp(0, 255);
    int g = (int.tryParse(gCtrl.text) ?? 0).clamp(0, 255);
    int b = (int.tryParse(bCtrl.text) ?? 0).clamp(0, 255);

    // Update controllers to reflect clamping if user typed e.g. 300
    _updateTextIfChanged(rCtrl, r.toString());
    _updateTextIfChanged(gCtrl, g.toString());
    _updateTextIfChanged(bCtrl, b.toString());

    Color newColor = Color.fromARGB(255, r, g, b);
    HSVColor newHsv = HSVColor.fromColor(newColor);

    if (newHsv != hsvColor) {
      setState(() {
        hsvColor = newHsv;
      });
      widget.onColorChanged(newColor);
    }
  }

  @override
  Widget build(BuildContext context) {
    final currentColor = hsvColor.toColor();
    return SizedBox(
      width: 340,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          // Upper Area: Area + Slider
          Row(
            children: [
              // 2D Saturation/Value Picker
              Expanded(
                child: Container(
                  height: 200,
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: Colors.white10),
                  ),
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(11),
                    child: ColorPickerArea(
                      hsvColor,
                      _updateColor,
                      PaletteType.hsv,
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 16),
              // Vertical Hue Slider
              _buildHueSlider(200),
            ],
          ),
          const SizedBox(height: 24),
          // Lower Area: RGB Fields + Preview
          Row(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              _buildRGBField('RED', rCtrl),
              const SizedBox(width: 10),
              _buildRGBField('GREEN', gCtrl),
              const SizedBox(width: 10),
              _buildRGBField('BLUE', bCtrl),
              const SizedBox(width: 20),
              // NEW: Color Preview Circle
              LedPreview(color: currentColor, size: 50),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildHueSlider(double height) {
    return GestureDetector(
      onVerticalDragUpdate: (details) {
        double newHue = (details.localPosition.dy / height) * 360;
        _updateColor(hsvColor.withHue(newHue.clamp(0.0, 360.0)));
      },
      onTapDown: (details) {
        double newHue = (details.localPosition.dy / height) * 360;
        _updateColor(hsvColor.withHue(newHue.clamp(0.0, 360.0)));
      },
      child: Container(
        width: 30,
        height: height,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(15),
          gradient: const LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [
              Color(0xFFFF0000),
              Color(0xFFFFFF00),
              Color(0xFF00FF00),
              Color(0xFF00FFFF),
              Color(0xFF0000FF),
              Color(0xFFFF00FF),
              Color(0xFFFF0000),
            ],
          ),
        ),
        child: Stack(
          alignment: Alignment.topCenter,
          clipBehavior: Clip.none,
          children: [
            Positioned(
              top: (hsvColor.hue / 360) * height - 3,
              child: Container(
                height: 6,
                width: 36,
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(3),
                  boxShadow: const [
                    BoxShadow(blurRadius: 3, color: Colors.black54),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildRGBField(String label, TextEditingController ctrl) {
    return Expanded(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: const TextStyle(
              fontSize: 10,
              fontWeight: FontWeight.bold,
              color: Colors.white54,
              letterSpacing: 0.5,
            ),
          ),
          const SizedBox(height: 6),
          SizedBox(
            height: 40,
            child: TextFormField(
              controller: ctrl,
              keyboardType: TextInputType.number,
              textAlign: TextAlign.center,
              style: const TextStyle(
                fontSize: 14,
                fontFamily: 'monospace',
                fontWeight: FontWeight.bold,
              ),
              decoration: InputDecoration(
                contentPadding: EdgeInsets.zero,
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(
                    color: Colors.deepPurpleAccent,
                    width: 2,
                  ),
                ),
              ),
              onChanged: (_) => _updateFromText(),
            ),
          ),
        ],
      ),
    );
  }
}
