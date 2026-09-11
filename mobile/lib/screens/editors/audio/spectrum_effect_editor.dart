import 'package:flutter/material.dart';
import '../../../models.dart';
import '../../../utils.dart';
import '../../../widgets/color_picker_tile.dart';
import '../../../widgets/section_header.dart';

class AudioSpectrumEffectEditor extends StatefulWidget {
  final AudioSpectrumConfig initialConfig;

  const AudioSpectrumEffectEditor({
    super.key,
    required this.initialConfig,
  });

  @override
  State<AudioSpectrumEffectEditor> createState() =>
      _AudioSpectrumEffectEditorState();
}

class _AudioSpectrumEffectEditorState extends State<AudioSpectrumEffectEditor> {
  late Color ledLow;
  late Color ledMid;
  late Color ledHigh;

  @override
  void initState() {
    super.initState();
    final spec = widget.initialConfig;
    ledLow = fromRgbList(spec.ledLow);
    ledMid = fromRgbList(spec.ledMid);
    ledHigh = fromRgbList(spec.ledHigh);
  }

  void _done() {
    final updated = AudioSpectrumConfig(
      ledLow: toRgbList(ledLow),
      ledMid: toRgbList(ledMid),
      ledHigh: toRgbList(ledHigh),
    );
    Navigator.pop(context, updated);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Spectrum Analyzer Effect'),
        actions: [
          IconButton(
            icon: const Icon(Icons.check),
            tooltip: 'Apply',
            onPressed: _done,
          ),
        ],
      ),
      body: PopScope(
        canPop: false,
        onPopInvokedWithResult: (didPop, result) {
          if (didPop) return;
          _done();
        },
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Card(
              color: const Color(0xFF1E1E1E),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(8),
                side: BorderSide(
                  color: Colors.greenAccent.withValues(alpha: 0.3),
                ),
              ),
              child: const Padding(
                padding: EdgeInsets.all(14),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Icon(Icons.info_outline, color: Colors.greenAccent, size: 22),
                    SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        'Spectrum effects render 16 frequency bands across configured LED segments. '
                        'Each band intensity interpolates linearly through the Low (0%), Mid (50%), and High (100%) gradient. '
                        'CPU-intensive FFT audio processing is dynamically enabled only while an active scene contains spectrum segments.',
                        style: TextStyle(fontSize: 13, height: 1.4, color: Colors.white70),
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 20),
            const SectionHeader('Spectrum Gradient Colors', color: Colors.greenAccent),
            ColorPickerTile(
              label: 'Low Energy Color (0%..50%)',
              color: ledLow,
              onColorChanged: (c) => setState(() => ledLow = c),
            ),
            ColorPickerTile(
              label: 'Mid Energy Color (50%)',
              color: ledMid,
              onColorChanged: (c) => setState(() => ledMid = c),
            ),
            ColorPickerTile(
              label: 'Peak Energy Color (50%..100%)',
              color: ledHigh,
              onColorChanged: (c) => setState(() => ledHigh = c),
            ),
          ],
        ),
      ),
    );
  }
}
