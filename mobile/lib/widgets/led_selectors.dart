import 'package:flutter/material.dart';

const double _kHorizontalPadding = 16.0;

class _SelectorKnob extends StatelessWidget {
  final double position;
  final Color color;
  final String label;
  final double fontSize;
  final ValueChanged<double> onDrag;

  const _SelectorKnob({
    required this.position,
    required this.color,
    required this.label,
    this.fontSize = 10,
    required this.onDrag,
  });

  @override
  Widget build(BuildContext context) {
    return Positioned(
      left: position - 14,
      child: GestureDetector(
        onHorizontalDragUpdate: (details) => onDrag(details.delta.dx),
        child: Container(
          width: 28,
          height: 28,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: color,
            shape: BoxShape.circle,
            border: Border.all(color: Colors.white, width: 2),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: 0.5),
                blurRadius: 4,
                offset: const Offset(0, 2),
              ),
            ],
          ),
          child: Text(
            label,
            style: TextStyle(
              color: Colors.black,
              fontWeight: FontWeight.bold,
              fontSize: fontSize,
            ),
          ),
        ),
      ),
    );
  }
}

class LedPointSelector extends StatelessWidget {
  final String label;
  final int value;
  final int totalLeds;
  final ValueChanged<int> onChanged;
  final Color color;

  const LedPointSelector({
    super.key,
    required this.label,
    required this.value,
    required this.totalLeds,
    required this.onChanged,
    this.color = Colors.blueAccent,
  });

  void _showEditDialog(BuildContext context) {
    final controller = TextEditingController(text: value.toString());

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Set $label', style: const TextStyle(fontSize: 16)),
        content: SizedBox(
          width: 200,
          child: TextField(
            controller: controller,
            autofocus: true,
            keyboardType: TextInputType.number,
            decoration: const InputDecoration(
              labelText: 'Value',
              floatingLabelBehavior: FloatingLabelBehavior.always,
              border: OutlineInputBorder(),
            ),
            onSubmitted: (text) {
              final parsed = int.tryParse(text);
              if (parsed != null) {
                onChanged(parsed.clamp(0, totalLeds - 1));
              }
              Navigator.pop(ctx);
            },
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () {
              final parsed = int.tryParse(controller.text);
              if (parsed != null) {
                onChanged(parsed.clamp(0, totalLeds - 1));
              }
              Navigator.pop(ctx);
            },
            child: const Text('OK'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label, style: const TextStyle(fontWeight: FontWeight.w500)),
            InkWell(
              onTap: () => _showEditDialog(context),
              borderRadius: BorderRadius.circular(4),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
                child: Text(
                  'LED #$value',
                  style: const TextStyle(
                    fontFamily: 'monospace',
                    fontWeight: FontWeight.bold,
                    decoration: TextDecoration.underline,
                    decorationStyle: TextDecorationStyle.dotted,
                  ),
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        LayoutBuilder(
          builder: (context, constraints) {
            final double width = constraints.maxWidth;
            final double usableWidth = width - (2 * _kHorizontalPadding);
            final double knobPos =
                (value / (totalLeds - 1)) * usableWidth + _kHorizontalPadding;

            void updatePos(double localX) {
              final double clampedX = localX.clamp(
                _kHorizontalPadding,
                width - _kHorizontalPadding,
              );
              final int newValue =
                  (((clampedX - _kHorizontalPadding) / usableWidth) *
                          (totalLeds - 1))
                      .round()
                      .clamp(0, totalLeds - 1);
              onChanged(newValue);
            }

            return GestureDetector(
              onHorizontalDragUpdate: (details) =>
                  updatePos(knobPos + details.delta.dx),
              onTapUp: (details) => updatePos(details.localPosition.dx),
              child: SizedBox(
                height: 30,
                child: Stack(
                  alignment: Alignment.centerLeft,
                  clipBehavior: Clip.none,
                  children: [
                    // Track
                    Positioned(
                      left: _kHorizontalPadding,
                      right: _kHorizontalPadding,
                      child: Container(
                        height: 4,
                        decoration: BoxDecoration(
                          color: Colors.grey.shade800,
                          borderRadius: BorderRadius.circular(2),
                        ),
                      ),
                    ),
                    // Knob
                    Positioned(
                      left: knobPos - 12,
                      child: Container(
                        width: 24,
                        height: 24,
                        decoration: BoxDecoration(
                          color: color,
                          shape: BoxShape.circle,
                          boxShadow: [
                            BoxShadow(
                              color: color.withValues(alpha: 0.4),
                              blurRadius: 6,
                              spreadRadius: 2,
                            ),
                          ],
                          border: Border.all(color: Colors.white, width: 2),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            );
          },
        ),
      ],
    );
  }
}

class LedRangeSelector extends StatelessWidget {
  final String label;
  final int start;
  final int end;
  final int totalLeds;
  final Function(int start, int end) onChanged;

  const LedRangeSelector({
    super.key,
    required this.label,
    required this.start,
    required this.end,
    required this.totalLeds,
    required this.onChanged,
  });

  void _showEditDialog(BuildContext context) {
    final startCtrl = TextEditingController(text: start.toString());
    final endCtrl = TextEditingController(text: end.toString());

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Set $label', style: const TextStyle(fontSize: 16)),
        content: SizedBox(
          width: 260,
          child: Row(
            children: [
              Expanded(
                child: TextField(
                  controller: startCtrl,
                  autofocus: true,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(
                    labelText: 'Start',
                    floatingLabelBehavior: FloatingLabelBehavior.always,
                    border: OutlineInputBorder(),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: TextField(
                  controller: endCtrl,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(
                    labelText: 'End',
                    floatingLabelBehavior: FloatingLabelBehavior.always,
                    border: OutlineInputBorder(),
                  ),
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () {
              final newStart = int.tryParse(startCtrl.text);
              final newEnd = int.tryParse(endCtrl.text);
              if (newStart != null && newEnd != null) {
                onChanged(
                  newStart.clamp(0, totalLeds - 1),
                  newEnd.clamp(0, totalLeds - 1),
                );
              }
              Navigator.pop(ctx);
            },
            child: const Text('OK'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label, style: const TextStyle(fontWeight: FontWeight.w500)),
            InkWell(
              onTap: () => _showEditDialog(context),
              borderRadius: BorderRadius.circular(4),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
                child: Text(
                  '$start \u2192 $end',
                  style: TextStyle(
                    fontFamily: 'monospace',
                    fontWeight: FontWeight.bold,
                    color: start <= end ? Colors.white : Colors.orangeAccent,
                    decoration: TextDecoration.underline,
                    decorationStyle: TextDecorationStyle.dotted,
                  ),
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        LayoutBuilder(
          builder: (context, constraints) {
            final double width = constraints.maxWidth;
            final double usableWidth = width - (2 * _kHorizontalPadding);

            double toPos(int val) =>
                (val / (totalLeds - 1)) * usableWidth + _kHorizontalPadding;
            int toVal(double pos) =>
                (((pos - _kHorizontalPadding) / usableWidth) * (totalLeds - 1))
                    .round()
                    .clamp(0, totalLeds - 1);

            final double startPos = toPos(start);
            final double endPos = toPos(end);

            return SizedBox(
              height: 40,
              child: Stack(
                alignment: Alignment.centerLeft,
                clipBehavior: Clip.none,
                children: [
                  // Track Background
                  Positioned(
                    left: _kHorizontalPadding,
                    right: _kHorizontalPadding,
                    child: Container(
                      height: 6,
                      decoration: BoxDecoration(
                        color: Colors.grey.shade800,
                        borderRadius: BorderRadius.circular(3),
                      ),
                    ),
                  ),

                  // Active Segment
                  Positioned(
                    left: startPos <= endPos ? startPos : endPos,
                    width: (endPos - startPos).abs(),
                    child: Container(
                      height: 6,
                      decoration: BoxDecoration(
                        gradient: LinearGradient(
                          colors: startPos <= endPos
                              ? [
                                  Colors.greenAccent.withValues(alpha: 0.5),
                                  Colors.redAccent.withValues(alpha: 0.5),
                                ]
                              : [
                                  Colors.redAccent.withValues(alpha: 0.5),
                                  Colors.greenAccent.withValues(alpha: 0.5),
                                ],
                        ),
                      ),
                    ),
                  ),

                  // Start Knob
                  _SelectorKnob(
                    position: startPos,
                    color: Colors.greenAccent,
                    label: 'S',
                    onDrag: (dx) {
                      final double newPos = (startPos + dx).clamp(
                        _kHorizontalPadding,
                        width - _kHorizontalPadding,
                      );
                      onChanged(toVal(newPos), end);
                    },
                  ),

                  // End Knob
                  _SelectorKnob(
                    position: endPos,
                    color: Colors.redAccent,
                    label: 'E',
                    onDrag: (dx) {
                      final double newPos = (endPos + dx).clamp(
                        _kHorizontalPadding,
                        width - _kHorizontalPadding,
                      );
                      onChanged(start, toVal(newPos));
                    },
                  ),
                ],
              ),
            );
          },
        ),
      ],
    );
  }
}

class DbRangeSelector extends StatelessWidget {
  final String label;
  final double minDb;
  final double maxDb;
  final double rangeMin;
  final double rangeMax;
  final Function(double min, double max) onChanged;

  const DbRangeSelector({
    super.key,
    required this.label,
    required this.minDb,
    required this.maxDb,
    this.rangeMin = -90.0,
    this.rangeMax = 0.0,
    required this.onChanged,
  });

  void _showEditDialog(BuildContext context) {
    final minCtrl = TextEditingController(text: minDb.toStringAsFixed(1));
    final maxCtrl = TextEditingController(text: maxDb.toStringAsFixed(1));

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Set $label', style: const TextStyle(fontSize: 16)),
        content: SizedBox(
          width: 260,
          child: Row(
            children: [
              Expanded(
                child: TextField(
                  controller: minCtrl,
                  autofocus: true,
                  keyboardType: const TextInputType.numberWithOptions(
                    decimal: true,
                    signed: true,
                  ),
                  decoration: const InputDecoration(
                    labelText: 'Start',
                    floatingLabelBehavior: FloatingLabelBehavior.always,
                    suffixText: 'dB',
                    border: OutlineInputBorder(),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: TextField(
                  controller: maxCtrl,
                  keyboardType: const TextInputType.numberWithOptions(
                    decimal: true,
                    signed: true,
                  ),
                  decoration: const InputDecoration(
                    labelText: 'End',
                    floatingLabelBehavior: FloatingLabelBehavior.always,
                    suffixText: 'dB',
                    border: OutlineInputBorder(),
                  ),
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () {
              double? newMin = double.tryParse(minCtrl.text);
              double? newMax = double.tryParse(maxCtrl.text);
              if (newMin != null && newMax != null) {
                newMin = newMin.clamp(rangeMin, rangeMax);
                newMax = newMax.clamp(rangeMin, rangeMax);
                if (newMin >= newMax) {
                  newMin = newMax - 0.5;
                }
                onChanged(newMin, newMax);
              }
              Navigator.pop(ctx);
            },
            child: const Text('OK'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label, style: const TextStyle(fontWeight: FontWeight.w500)),
            InkWell(
              onTap: () => _showEditDialog(context),
              borderRadius: BorderRadius.circular(4),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
                child: Text(
                  '${minDb.toStringAsFixed(1)} dB  \u2194  ${maxDb.toStringAsFixed(1)} dB',
                  style: const TextStyle(
                    fontFamily: 'monospace',
                    fontWeight: FontWeight.bold,
                    decoration: TextDecoration.underline,
                    decorationStyle: TextDecorationStyle.dotted,
                  ),
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        LayoutBuilder(
          builder: (context, constraints) {
            final double width = constraints.maxWidth;
            final double usableWidth = width - (2 * _kHorizontalPadding);
            final double rangeSpan = rangeMax - rangeMin;

            double toPos(double db) =>
                ((db - rangeMin) / rangeSpan) * usableWidth +
                _kHorizontalPadding;
            double toDb(double pos) =>
                ((pos - _kHorizontalPadding) / usableWidth) * rangeSpan +
                rangeMin;

            final double minPos = toPos(minDb);
            final double maxPos = toPos(maxDb);

            return SizedBox(
              height: 40,
              child: Stack(
                alignment: Alignment.centerLeft,
                clipBehavior: Clip.none,
                children: [
                  // Track
                  Positioned(
                    left: _kHorizontalPadding,
                    right: _kHorizontalPadding,
                    child: Container(
                      height: 6,
                      decoration: BoxDecoration(
                        color: Colors.grey.shade800,
                        borderRadius: BorderRadius.circular(3),
                      ),
                    ),
                  ),

                  // Active Range
                  Positioned(
                    left: minPos,
                    width: (maxPos - minPos).abs(),
                    child: Container(
                      height: 6,
                      color: Colors.greenAccent.withValues(alpha: 0.5),
                    ),
                  ),

                  // Min Knob
                  _SelectorKnob(
                    position: minPos,
                    color: Colors.greenAccent,
                    label: 'MIN',
                    fontSize: 8,
                    onDrag: (dx) {
                      final double newPos = (minPos + dx).clamp(
                        _kHorizontalPadding,
                        width - _kHorizontalPadding,
                      );
                      double newVal = toDb(newPos).clamp(rangeMin, rangeMax);
                      if (newVal >= maxDb) newVal = maxDb - 0.5;
                      onChanged(newVal, maxDb);
                    },
                  ),

                  // Max Knob
                  _SelectorKnob(
                    position: maxPos,
                    color: Colors.redAccent,
                    label: 'MAX',
                    fontSize: 8,
                    onDrag: (dx) {
                      final double newPos = (maxPos + dx).clamp(
                        _kHorizontalPadding,
                        width - _kHorizontalPadding,
                      );
                      double newVal = toDb(newPos).clamp(rangeMin, rangeMax);
                      if (newVal <= minDb) newVal = minDb + 0.5;
                      onChanged(minDb, newVal);
                    },
                  ),
                ],
              ),
            );
          },
        ),
      ],
    );
  }
}
