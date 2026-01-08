import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../widgets/config_slider.dart';
import '../../models.dart';
import '../../utils.dart';
import '../../widgets/rgb_input_picker.dart';
import '../../widgets/led_preview.dart';

class MultiBlobLEDEditor extends StatefulWidget {
  const MultiBlobLEDEditor({super.key});

  @override
  State<MultiBlobLEDEditor> createState() => _MultiBlobLEDEditorState();
}

class _MultiBlobLEDEditorState extends State<MultiBlobLEDEditor> {
  late int durationSec;
  late int delayMs;
  late List<BlobCfg> blobs;
  int ledsTotal = 100;

  bool _initialized = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_initialized) {
      final config = context.read<ConfigProvider>().config;
      if (config != null) {
        ledsTotal = config.ledsTotal;
        final m = config.multiBlobLED;
        durationSec = m.durationSec;
        delayMs = m.delayMs;
        // Deep copy blobs to allow editing/canceling
        blobs = m.blobCfg
            .map(
              (b) => BlobCfg(
                deltaX: b.deltaX,
                x: b.x,
                width: b.width,
                ledRGB: List.from(b.ledRGB),
              ),
            )
            .toList();
        _initialized = true;
      }
    }
  }

  void _save() {
    if (blobs.isEmpty) {
      showDialog(
        context: context,
        builder: (ctx) => AlertDialog(
          title: const Text('Warning'),
          content: const Text(
            'You are about to save an empty list of blobs.\n'
            'This will remove all existing blobs from the configuration.\n\n'
            'Are you sure you want to proceed?',
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel'),
            ),
            ElevatedButton(
              style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
              onPressed: () {
                Navigator.pop(ctx);
                _performSave();
              },
              child: const Text('Overwrite & Clear'),
            ),
          ],
        ),
      );
    } else {
      _performSave();
    }
  }

  void _performSave() {
    final provider = context.read<ConfigProvider>();
    final config = provider.config;
    if (config == null) return;

    config.multiBlobLED.durationSec = durationSec;
    config.multiBlobLED.delayMs = delayMs;
    config.multiBlobLED.blobCfg = blobs;

    provider.updateConfig(config).then((_) {
      if (mounted) Navigator.pop(context);
    });
  }

  void _editBlob({int? index}) {
    // If index is null, we are creating a new blob
    BlobCfg tempBlob;
    if (index != null) {
      final b = blobs[index];
      tempBlob = BlobCfg(
        deltaX: b.deltaX,
        x: b.x,
        width: b.width,
        ledRGB: List.from(b.ledRGB),
      );
    } else {
      tempBlob = BlobCfg(
        deltaX: 0.5,
        x: 0,
        width: 2,
        ledRGB: [0, 0, 255],
      ); // Default blue blob
    }

    Color tempColor = fromRgbList(tempBlob.ledRGB);

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (context, setState) {
          return AlertDialog(
            title: Text(index == null ? 'Add Blob' : 'Edit Blob'),
            content: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  ConfigSlider(
                    label: 'Speed (DeltaX)',
                    value: tempBlob.deltaX,
                    min: -2.0,
                    max: 2.0,
                    isInt: false,
                    onChanged: (v) => setState(
                      () =>
                          tempBlob.deltaX = double.parse(v.toStringAsFixed(2)),
                    ),
                    activeColor: Colors.pinkAccent,
                  ),
                  ConfigSlider(
                    label: 'Initial Pos (X)',
                    value: tempBlob.x,
                    min: 0,
                    max: ledsTotal.toDouble(),
                    onChanged: (v) => setState(() => tempBlob.x = v),
                    activeColor: Colors.pinkAccent,
                  ),
                  ConfigSlider(
                    label: 'Width',
                    value: tempBlob.width,
                    min: 1,
                    max: 1024,
                    onChanged: (v) => setState(() => tempBlob.width = v),
                    activeColor: Colors.pinkAccent,
                  ),
                  const SizedBox(height: 16),
                  const Text(
                    'Blob Color',
                    style: TextStyle(fontWeight: FontWeight.bold),
                  ),
                  const SizedBox(height: 8),
                  RgbInputPicker(
                    initialColor: tempColor,
                    onColorChanged: (c) {
                      tempColor = c;
                      tempBlob.ledRGB = toRgbList(c);
                    },
                  ),
                ],
              ),
            ),
            actions: [
              if (index != null)
                TextButton(
                  onPressed: () {
                    this.setState(() => blobs.removeAt(index));
                    Navigator.pop(ctx);
                  },
                  child: const Text(
                    'DELETE',
                    style: TextStyle(color: Colors.red),
                  ),
                ),
              ElevatedButton(
                onPressed: () {
                  this.setState(() {
                    if (index != null) {
                      blobs[index] = tempBlob;
                    } else {
                      blobs.add(tempBlob);
                    }
                  });
                  Navigator.pop(ctx);
                },
                child: const Text('DONE'),
              ),
            ],
          );
        },
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    if (!_initialized)
      return const Scaffold(body: Center(child: CircularProgressIndicator()));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Multi Blob Config'),
        actions: [IconButton(icon: const Icon(Icons.save), onPressed: _save)],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSectionHeader('Global Settings'),
          ConfigSlider(
            label: 'Cycle Duration',
            value: durationSec.toDouble(),
            min: 10,
            max: 300,
            unit: 's',
            onChanged: (v) => setState(() => durationSec = v.toInt()),
            activeColor: Colors.pinkAccent,
          ),
          ConfigSlider(
            label: 'Step Delay',
            value: delayMs.toDouble(),
            min: 10,
            max: 200,
            unit: 'ms',
            onChanged: (v) => setState(() => delayMs = v.toInt()),
            activeColor: Colors.pinkAccent,
          ),

          const SizedBox(height: 24),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              _buildSectionHeader('Blobs'),
              IconButton(
                icon: const Icon(Icons.add_circle, color: Colors.pinkAccent),
                onPressed: () => _editBlob(),
              ),
            ],
          ),
          ...blobs.asMap().entries.map((entry) {
            final i = entry.key;
            final b = entry.value;
            final color = fromRgbList(b.ledRGB);
            return Card(
              color: Colors.grey.shade900,
              margin: const EdgeInsets.only(bottom: 8),
              child: ListTile(
                leading: LedPreview(color: color, size: 24),
                title: Text('Blob $i (Width: ${b.width.toInt()})'),
                subtitle: Text(
                  'Pos: ${b.x.toInt()}, Speed: ${b.deltaX.toStringAsFixed(2)}',
                ),
                trailing: const Icon(Icons.edit, size: 20),
                onTap: () => _editBlob(index: i),
              ),
            );
          }),
        ],
      ),
    );
  }

  Widget _buildSectionHeader(String title) {
    return Text(
      title.toUpperCase(),
      style: const TextStyle(
        color: Colors.pinkAccent,
        fontWeight: FontWeight.bold,
        letterSpacing: 1.2,
      ),
    );
  }
}
