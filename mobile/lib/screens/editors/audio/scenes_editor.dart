import 'dart:math';
import 'package:flutter/material.dart';
import '../../../models.dart';
import '../../../widgets/led_selectors.dart';

const List<Map<String, String>> kAudioEffects = [
  {'value': 'LeftVU', 'label': 'LeftVU (Left Channel VU Meter)'},
  {'value': 'RightVU', 'label': 'RightVU (Right Channel VU Meter)'},
  {'value': 'MonoVU', 'label': 'MonoVU (Mono Average VU Meter)'},
  {'value': 'LeftSpectrum', 'label': 'LeftSpectrum (16-Band Left Spectrum)'},
  {'value': 'RightSpectrum', 'label': 'RightSpectrum (16-Band Right Spectrum)'},
  {'value': 'MonoSpectrum', 'label': 'MonoSpectrum (16-Band Mono Spectrum)'},
];

bool isSpectrumEffect(String eff) {
  return eff == 'LeftSpectrum' ||
      eff == 'RightSpectrum' ||
      eff == 'MonoSpectrum' ||
      eff == 'Spectrum';
}

class AudioScenesResult {
  final List<AudioSceneConfig> scenes;
  final String activeScene;

  const AudioScenesResult({
    required this.scenes,
    required this.activeScene,
  });
}

class AudioScenesEditor extends StatefulWidget {
  final List<AudioSceneConfig> initialScenes;
  final String initialActiveScene;
  final int totalLeds;

  const AudioScenesEditor({
    super.key,
    required this.initialScenes,
    required this.initialActiveScene,
    required this.totalLeds,
  });

  @override
  State<AudioScenesEditor> createState() => _AudioScenesEditorState();
}

class _AudioScenesEditorState extends State<AudioScenesEditor> {
  late List<AudioSceneConfig> scenes;
  late String activeScene;
  int expandedSceneIndex = 0;

  @override
  void initState() {
    super.initState();
    scenes = widget.initialScenes
        .map(
          (s) => AudioSceneConfig(
            name: s.name,
            segments: s.segments.map((seg) => seg.copyWith()).toList(),
          ),
        )
        .toList();
    activeScene = widget.initialActiveScene;

    if (scenes.isNotEmpty && !scenes.any((s) => s.name == activeScene)) {
      activeScene = scenes.first.name;
    }
  }

  void _done() {
    // If activeScene not valid, pick first
    if (scenes.isNotEmpty && !scenes.any((s) => s.name == activeScene)) {
      activeScene = scenes.first.name;
    }
    Navigator.pop(
      context,
      AudioScenesResult(scenes: scenes, activeScene: activeScene),
    );
  }

  void _addScene() {
    final ctrl = TextEditingController(text: 'Scene ${scenes.length + 1}');
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('New Scene'),
        content: TextField(
          controller: ctrl,
          autofocus: true,
          decoration: const InputDecoration(
            labelText: 'Scene Name',
            border: OutlineInputBorder(),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () {
              final name = ctrl.text.trim();
              if (name.isNotEmpty) {
                setState(() {
                  scenes.add(
                    AudioSceneConfig(
                      name: name,
                      segments: [
                        AudioSegmentConfig(
                          startLed: 0,
                          endLed: min(19, widget.totalLeds - 1),
                          effect: 'MonoSpectrum',
                        ),
                      ],
                    ),
                  );
                  expandedSceneIndex = scenes.length - 1;
                });
              }
              Navigator.pop(ctx);
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }

  void _renameScene(int index) {
    final ctrl = TextEditingController(text: scenes[index].name);
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Rename Scene'),
        content: TextField(
          controller: ctrl,
          autofocus: true,
          decoration: const InputDecoration(
            labelText: 'Scene Name',
            border: OutlineInputBorder(),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () {
              final newName = ctrl.text.trim();
              if (newName.isNotEmpty) {
                final oldName = scenes[index].name;
                setState(() {
                  scenes[index] = scenes[index].copyWith(name: newName);
                  if (activeScene == oldName) {
                    activeScene = newName;
                  }
                });
              }
              Navigator.pop(ctx);
            },
            child: const Text('Rename'),
          ),
        ],
      ),
    );
  }

  void _deleteScene(int index) {
    if (scenes.length <= 1) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('At least one scene must remain.')),
      );
      return;
    }

    final scene = scenes[index];
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Scene'),
        content: Text('Are you sure you want to delete "${scene.name}"?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () {
              setState(() {
                scenes.removeAt(index);
                if (activeScene == scene.name) {
                  activeScene = scenes.first.name;
                }
                if (expandedSceneIndex >= scenes.length) {
                  expandedSceneIndex = scenes.length - 1;
                }
              });
              Navigator.pop(ctx);
            },
            child: const Text('Delete'),
          ),
        ],
      ),
    );
  }

  void _addSegment(int sceneIdx) {
    final scene = scenes[sceneIdx];
    int maxEnd = -1;
    for (final s in scene.segments) {
      final m = max(s.startLed, s.endLed);
      if (m > maxEnd) maxEnd = m;
    }

    int newStart = maxEnd + 1;
    if (newStart >= widget.totalLeds) newStart = 0;
    int newEnd = min(newStart + 19, widget.totalLeds - 1);

    setState(() {
      final updatedSegments = List<AudioSegmentConfig>.from(scene.segments)
        ..add(
          AudioSegmentConfig(
            startLed: newStart,
            endLed: newEnd,
            effect: 'MonoSpectrum',
          ),
        );
      scenes[sceneIdx] = scene.copyWith(segments: updatedSegments);
    });
  }

  void _deleteSegment(int sceneIdx, int segIdx) {
    final scene = scenes[sceneIdx];
    if (scene.segments.length <= 1) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('A scene must have at least one segment.')),
      );
      return;
    }

    setState(() {
      final updatedSegments = List<AudioSegmentConfig>.from(scene.segments)
        ..removeAt(segIdx);
      scenes[sceneIdx] = scene.copyWith(segments: updatedSegments);
    });
  }

  bool _hasSegmentOverlap(List<AudioSegmentConfig> segs) {
    final used = <int>{};
    for (final seg in segs) {
      final s = min(seg.startLed, seg.endLed);
      final e = max(seg.startLed, seg.endLed);
      for (int i = s; i <= e; i++) {
        if (used.contains(i)) return true;
        used.add(i);
      }
    }
    return false;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Audio Scenes Manager'),
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            tooltip: 'Add Scene',
            onPressed: _addScene,
          ),
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
        child: scenes.isEmpty
            ? Center(
                child: ElevatedButton.icon(
                  icon: const Icon(Icons.add),
                  label: const Text('Create Scene'),
                  onPressed: _addScene,
                ),
              )
            : ListView.builder(
                padding: const EdgeInsets.all(16),
                itemCount: scenes.length,
                itemBuilder: (context, sIdx) {
                  final scene = scenes[sIdx];
                  final isActive = scene.name == activeScene;
                  final isExpanded = sIdx == expandedSceneIndex;
                  final hasOverlap = _hasSegmentOverlap(scene.segments);

                  return Card(
                    margin: const EdgeInsets.only(bottom: 16),
                    color: const Color(0xFF1E1E1E),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(10),
                      side: BorderSide(
                        color: isActive
                            ? Colors.greenAccent
                            : Colors.white.withValues(alpha: 0.12),
                        width: isActive ? 2.0 : 1.0,
                      ),
                    ),
                    child: Column(
                      children: [
                        // Scene Header
                        ListTile(
                          contentPadding: const EdgeInsets.symmetric(
                            horizontal: 16,
                            vertical: 4,
                          ),
                          title: Row(
                            children: [
                              Text(
                                scene.name,
                                style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  fontSize: 16,
                                  color: isActive
                                      ? Colors.greenAccent
                                      : Colors.white,
                                ),
                              ),
                              const SizedBox(width: 8),
                              if (isActive)
                                Container(
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 8,
                                    vertical: 2,
                                  ),
                                  decoration: BoxDecoration(
                                    color: Colors.greenAccent.withValues(alpha: 0.2),
                                    borderRadius: BorderRadius.circular(4),
                                  ),
                                  child: const Text(
                                    'ACTIVE',
                                    style: TextStyle(
                                      color: Colors.greenAccent,
                                      fontSize: 11,
                                      fontWeight: FontWeight.bold,
                                    ),
                                  ),
                                ),
                            ],
                          ),
                          subtitle: Text(
                            '${scene.segments.length} segment${scene.segments.length == 1 ? '' : 's'}'
                            '${hasOverlap ? ' \u2022 \u26A0 Overlapping LEDs' : ''}',
                            style: TextStyle(
                              color: hasOverlap
                                  ? Colors.orangeAccent
                                  : Colors.grey,
                            ),
                          ),
                          trailing: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              if (!isActive)
                                IconButton(
                                  icon: const Icon(
                                    Icons.check_circle_outline,
                                    color: Colors.grey,
                                  ),
                                  tooltip: 'Set as active scene',
                                  onPressed: () =>
                                      setState(() => activeScene = scene.name),
                                ),
                              IconButton(
                                icon: const Icon(Icons.edit, size: 20),
                                tooltip: 'Rename Scene',
                                onPressed: () => _renameScene(sIdx),
                              ),
                              IconButton(
                                icon: const Icon(Icons.delete_outline, size: 20),
                                tooltip: 'Delete Scene',
                                onPressed: () => _deleteScene(sIdx),
                              ),
                              Icon(
                                isExpanded
                                    ? Icons.expand_less
                                    : Icons.expand_more,
                                color: Colors.grey,
                              ),
                            ],
                          ),
                          onTap: () => setState(() {
                            expandedSceneIndex = isExpanded ? -1 : sIdx;
                          }),
                        ),

                        // Expandable Segment Editor List
                        if (isExpanded) ...[
                          const Divider(height: 1),
                          if (hasOverlap)
                            Container(
                              width: double.infinity,
                              color: Colors.orangeAccent.withValues(alpha: 0.15),
                              padding: const EdgeInsets.symmetric(
                                horizontal: 16,
                                vertical: 8,
                              ),
                              child: const Row(
                                children: [
                                  Icon(
                                    Icons.warning_amber_rounded,
                                    color: Colors.orangeAccent,
                                    size: 18,
                                  ),
                                  SizedBox(width: 8),
                                  Expanded(
                                    child: Text(
                                      'Overlapping segments detected in this scene. Segments must not share LEDs.',
                                      style: TextStyle(
                                        color: Colors.orangeAccent,
                                        fontSize: 12,
                                      ),
                                    ),
                                  ),
                                ],
                              ),
                            ),
                          ListView.separated(
                            shrinkWrap: true,
                            physics: const NeverScrollableScrollPhysics(),
                            padding: const EdgeInsets.all(12),
                            itemCount: scene.segments.length,
                            separatorBuilder: (_, _) =>
                                const Divider(height: 24),
                            itemBuilder: (context, segIdx) {
                              final seg = scene.segments[segIdx];
                              final segLen =
                                  max(seg.startLed, seg.endLed) -
                                  min(seg.startLed, seg.endLed) +
                                  1;
                              final isSpec = isSpectrumEffect(seg.effect);
                              final isTooShort = isSpec && segLen < 16;

                              return Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Row(
                                    mainAxisAlignment:
                                        MainAxisAlignment.spaceBetween,
                                    children: [
                                      Text(
                                        'Segment ${segIdx + 1}',
                                        style: const TextStyle(
                                          fontWeight: FontWeight.bold,
                                          fontSize: 14,
                                        ),
                                      ),
                                      Row(
                                        children: [
                                          Container(
                                            padding: const EdgeInsets.symmetric(
                                              horizontal: 6,
                                              vertical: 2,
                                            ),
                                            decoration: BoxDecoration(
                                              color: isTooShort
                                                  ? Colors.redAccent
                                                        .withValues(alpha: 0.2)
                                                  : Colors.white10,
                                              borderRadius:
                                                  BorderRadius.circular(4),
                                            ),
                                            child: Text(
                                              '$segLen LEDs${seg.startLed > seg.endLed ? ' (Rev)' : ''}',
                                              style: TextStyle(
                                                fontSize: 12,
                                                color: isTooShort
                                                  ? Colors.redAccent
                                                  : Colors.white70,
                                                fontWeight: isTooShort
                                                  ? FontWeight.bold
                                                  : FontWeight.normal,
                                              ),
                                            ),
                                          ),
                                          const SizedBox(width: 8),
                                          IconButton(
                                            icon: const Icon(
                                              Icons.remove_circle_outline,
                                              color: Colors.redAccent,
                                              size: 20,
                                            ),
                                            tooltip: 'Remove Segment',
                                            onPressed: () =>
                                                _deleteSegment(sIdx, segIdx),
                                          ),
                                        ],
                                      ),
                                    ],
                                  ),
                                  if (isTooShort)
                                    const Padding(
                                      padding: EdgeInsets.only(bottom: 8),
                                      child: Text(
                                        '\u26A0 Spectrum effects require at least 16 LEDs',
                                        style: TextStyle(
                                          color: Colors.redAccent,
                                          fontSize: 12,
                                        ),
                                      ),
                                    ),
                                  const SizedBox(height: 8),
                                  DropdownButtonFormField<String>(
                                    initialValue: kAudioEffects.any(
                                      (e) => e['value'] == seg.effect,
                                    )
                                        ? (seg.effect == 'Spectrum'
                                              ? 'MonoSpectrum'
                                              : seg.effect)
                                        : 'MonoSpectrum',
                                    decoration: const InputDecoration(
                                      labelText: 'Effect Type',
                                      border: OutlineInputBorder(),
                                      contentPadding: EdgeInsets.symmetric(
                                        horizontal: 12,
                                        vertical: 10,
                                      ),
                                    ),
                                    items: kAudioEffects.map((e) {
                                      return DropdownMenuItem<String>(
                                        value: e['value'],
                                        child: Text(e['label']!),
                                      );
                                    }).toList(),
                                    onChanged: (newEff) {
                                      if (newEff != null) {
                                        setState(() {
                                          final updatedSegs =
                                              List<AudioSegmentConfig>.from(
                                                scene.segments,
                                              );
                                          updatedSegs[segIdx] = seg.copyWith(
                                            effect: newEff,
                                          );
                                          scenes[sIdx] = scene.copyWith(
                                            segments: updatedSegs,
                                          );
                                        });
                                      }
                                    },
                                  ),
                                  const SizedBox(height: 12),
                                  LedRangeSelector(
                                    label: 'LED Range',
                                    start: seg.startLed,
                                    end: seg.endLed,
                                    totalLeds: widget.totalLeds,
                                    onChanged: (s, e) {
                                      setState(() {
                                        final updatedSegs =
                                            List<AudioSegmentConfig>.from(
                                              scene.segments,
                                            );
                                        updatedSegs[segIdx] = seg.copyWith(
                                          startLed: s,
                                          endLed: e,
                                        );
                                        scenes[sIdx] = scene.copyWith(
                                          segments: updatedSegs,
                                        );
                                      });
                                    },
                                  ),
                                ],
                              );
                            },
                          ),
                          Padding(
                            padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
                            child: OutlinedButton.icon(
                              style: OutlinedButton.styleFrom(
                                foregroundColor: Colors.greenAccent,
                                side: const BorderSide(
                                  color: Colors.greenAccent,
                                ),
                                minimumSize: const Size.fromHeight(40),
                              ),
                              icon: const Icon(Icons.add, size: 18),
                              label: const Text('Add Segment'),
                              onPressed: () => _addSegment(sIdx),
                            ),
                          ),
                        ],
                      ],
                    ),
                  );
                },
              ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: _addScene,
        tooltip: 'Add Scene',
        backgroundColor: Colors.greenAccent,
        foregroundColor: Colors.black,
        child: const Icon(Icons.add),
      ),
    );
  }
}
