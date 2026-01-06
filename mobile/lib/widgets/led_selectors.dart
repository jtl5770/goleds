import 'package:flutter/material.dart';

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

  @override
  Widget build(BuildContext context) {
    const double horizontalPadding = 16.0;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label, style: const TextStyle(fontWeight: FontWeight.w500)),
            Text(
              'LED #$value',
              style: const TextStyle(
                fontFamily: 'monospace',
                fontWeight: FontWeight.bold,
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        LayoutBuilder(
          builder: (context, constraints) {
            final double width = constraints.maxWidth;
            final double usableWidth = width - (2 * horizontalPadding);
            final double knobPos =
                (value / (totalLeds - 1)) * usableWidth + horizontalPadding;

            return GestureDetector(
              onHorizontalDragUpdate: (details) {
                double newPos = (knobPos + details.delta.dx).clamp(
                  horizontalPadding,
                  width - horizontalPadding,
                );
                int newValue =
                    (((newPos - horizontalPadding) / usableWidth) *
                            (totalLeds - 1))
                        .round()
                        .clamp(0, totalLeds - 1);
                onChanged(newValue);
              },
              onTapUp: (details) {
                double newPos = details.localPosition.dx.clamp(
                  horizontalPadding,
                  width - horizontalPadding,
                );
                int newValue =
                    (((newPos - horizontalPadding) / usableWidth) *
                            (totalLeds - 1))
                        .round()
                        .clamp(0, totalLeds - 1);
                onChanged(newValue);
              },
              child: SizedBox(
                height: 30,
                child: Stack(
                  alignment: Alignment.centerLeft,
                  clipBehavior: Clip.none,
                  children: [
                    // Track
                    Positioned(
                      left: horizontalPadding,
                      right: horizontalPadding,
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

  @override
  Widget build(BuildContext context) {
    const double horizontalPadding = 16.0;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label, style: const TextStyle(fontWeight: FontWeight.w500)),
            Text(
              '$start → $end',
              style: TextStyle(
                fontFamily: 'monospace',
                fontWeight: FontWeight.bold,
                color: start <= end ? Colors.white : Colors.orangeAccent,
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        LayoutBuilder(
          builder: (context, constraints) {
            final double width = constraints.maxWidth;
            final double usableWidth = width - (2 * horizontalPadding);

            double toPos(int val) =>
                (val / (totalLeds - 1)) * usableWidth + horizontalPadding;
            int toVal(double pos) =>
                (((pos - horizontalPadding) / usableWidth) * (totalLeds - 1))
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
                    left: horizontalPadding,
                    right: horizontalPadding,
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
                  _buildKnob(
                    position: startPos,
                    color: Colors.greenAccent,
                    label: 'S',
                    onDrag: (dx) {
                      double newPos = (startPos + dx).clamp(
                        horizontalPadding,
                        width - horizontalPadding,
                      );
                      onChanged(toVal(newPos), end);
                    },
                  ),

                  // End Knob
                  _buildKnob(
                    position: endPos,
                    color: Colors.redAccent,
                    label: 'E',
                    onDrag: (dx) {
                      double newPos = (endPos + dx).clamp(
                        horizontalPadding,
                        width - horizontalPadding,
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

  Widget _buildKnob({
    required double position,
    required Color color,
    required String label,
    required Function(double dx) onDrag,
  }) {
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
            style: const TextStyle(
              color: Colors.black,
              fontWeight: FontWeight.bold,
              fontSize: 10,
            ),
          ),
        ),
      ),
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

  @override
  Widget build(BuildContext context) {
    const double horizontalPadding = 16.0;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label, style: const TextStyle(fontWeight: FontWeight.w500)),
            Text(
              '${minDb.toStringAsFixed(1)} dB  ↔  ${maxDb.toStringAsFixed(1)} dB',
              style: const TextStyle(
                fontFamily: 'monospace',
                fontWeight: FontWeight.bold,
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        LayoutBuilder(
          builder: (context, constraints) {
            final double width = constraints.maxWidth;
            final double usableWidth = width - (2 * horizontalPadding);
            final double rangeSpan = rangeMax - rangeMin;

            double toPos(double db) =>
                ((db - rangeMin) / rangeSpan) * usableWidth + horizontalPadding;
            double toDb(double pos) =>
                ((pos - horizontalPadding) / usableWidth) * rangeSpan +
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
                    left: horizontalPadding,
                    right: horizontalPadding,
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
                  _buildKnob(
                    position: minPos,
                    color: Colors.greenAccent,
                    label: 'MIN',
                    onDrag: (dx) {
                      double newPos = (minPos + dx).clamp(
                        horizontalPadding,
                        width - horizontalPadding,
                      );
                      double newVal = toDb(newPos).clamp(rangeMin, rangeMax);
                      if (newVal >= maxDb) newVal = maxDb - 0.5;
                      onChanged(newVal, maxDb);
                    },
                  ),

                  // Max Knob
                  _buildKnob(
                    position: maxPos,
                    color: Colors.redAccent,
                    label: 'MAX',
                    onDrag: (dx) {
                      double newPos = (maxPos + dx).clamp(
                        horizontalPadding,
                        width - horizontalPadding,
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

  Widget _buildKnob({
    required double position,
    required Color color,
    required String label,
    required Function(double dx) onDrag,
  }) {
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
            style: const TextStyle(
              color: Colors.black,
              fontWeight: FontWeight.bold,
              fontSize: 8,
            ),
          ),
        ),
      ),
    );
  }
}
