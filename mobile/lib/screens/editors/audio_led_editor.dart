import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/config_provider.dart';
import '../../widgets/config_slider.dart';
import '../../widgets/led_selectors.dart';
import '../../widgets/section_header.dart';
import '../../models.dart';
import '../../utils.dart';
import 'audio/scenes_editor.dart';
import 'audio/vu_effect_editor.dart';
import 'audio/spectrum_effect_editor.dart';
import 'audio/squeezebox_editor.dart';

class AudioLEDEditor extends StatefulWidget {
  final AudioLEDConfig initialConfig;
  final int totalLeds;

  const AudioLEDEditor({
    super.key,
    required this.initialConfig,
    required this.totalLeds,
  });

  @override
  State<AudioLEDEditor> createState() => _AudioLEDEditorState();
}

class _AudioLEDEditorState extends State<AudioLEDEditor> {
  late SqueezeboxConfig squeezeboxConfig;
  late String activeScene;
  late List<AudioSceneConfig> scenes;
  late AudioVUConfig vu;
  late AudioSpectrumConfig spectrum;

  // Timing & Sensitivity
  late int updateFreqMs;
  late double minDB, maxDB;

  @override
  void initState() {
    super.initState();
    final a = widget.initialConfig;
    squeezeboxConfig = a.squeezebox;
    activeScene = a.activeScene;
    scenes = a.scenes.map((s) => s.copyWith()).toList();
    vu = a.vu.copyWith();
    spectrum = a.spectrum.copyWith();

    updateFreqMs = a.updateFreqMs > 0 ? a.updateFreqMs : 30;
    minDB = a.minDB;
    maxDB = a.maxDB;
  }

  void _save() {
    final provider = context.read<ConfigProvider>();
    final currentFullConfig = provider.config;
    if (currentFullConfig == null) return;

    if (scenes.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('At least one scene must be configured.')),
      );
      return;
    }

    final updatedAudioConfig = currentFullConfig.audioLED.copyWith(
      activeScene: activeScene,
      scenes: scenes,
      vu: vu,
      spectrum: spectrum,
      updateFreqMs: updateFreqMs,
      minDB: minDB,
      maxDB: maxDB,
      squeezebox: squeezeboxConfig,
    );

    provider
        .updateConfig(currentFullConfig.copyWith(audioLED: updatedAudioConfig))
        .then((_) {
          if (mounted) Navigator.pop(context);
        });
  }

  void _openScenesManager() async {
    final res = await Navigator.push<AudioScenesResult>(
      context,
      MaterialPageRoute(
        builder: (_) => AudioScenesEditor(
          initialScenes: scenes,
          initialActiveScene: activeScene,
          totalLeds: widget.totalLeds,
        ),
      ),
    );
    if (res != null) {
      setState(() {
        scenes = res.scenes;
        activeScene = res.activeScene;
      });
    }
  }

  void _openVUEffectEditor() async {
    final res = await Navigator.push<AudioVUConfig>(
      context,
      MaterialPageRoute(
        builder: (_) => AudioVUEffectEditor(initialConfig: vu),
      ),
    );
    if (res != null) {
      setState(() => vu = res);
    }
  }

  void _openSpectrumEffectEditor() async {
    final res = await Navigator.push<AudioSpectrumConfig>(
      context,
      MaterialPageRoute(
        builder: (_) => AudioSpectrumEffectEditor(initialConfig: spectrum),
      ),
    );
    if (res != null) {
      setState(() => spectrum = res);
    }
  }

  void _openSqueezeboxEditor() async {
    final res = await Navigator.push<SqueezeboxConfig>(
      context,
      MaterialPageRoute(
        builder: (_) => AudioSqueezeboxEditor(initialConfig: squeezeboxConfig),
      ),
    );
    if (res != null) {
      setState(() => squeezeboxConfig = res);
    }
  }

  Widget _buildColorSwatch(List<double> rgb) {
    final color = fromRgbList(rgb);
    return Container(
      width: 14,
      height: 14,
      margin: const EdgeInsets.only(left: 4),
      decoration: BoxDecoration(
        color: color,
        shape: BoxShape.circle,
        border: Border.all(color: Colors.white38, width: 1),
      ),
    );
  }

  Widget _buildNavCard({
    required IconData icon,
    required String title,
    required String subtitle,
    required VoidCallback onTap,
    Widget? trailingExtra,
  }) {
    return Card(
      color: const Color(0xFF1E1E1E),
      margin: const EdgeInsets.only(bottom: 12),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
      ),
      child: ListTile(
        leading: Icon(icon, color: Colors.greenAccent),
        title: Text(title, style: const TextStyle(fontWeight: FontWeight.bold)),
        subtitle: Text(
          subtitle,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(color: Colors.grey, fontSize: 12),
        ),
        trailing: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            ?trailingExtra,
            const SizedBox(width: 4),
            const Icon(Icons.chevron_right, color: Colors.grey),
          ],
        ),
        onTap: onTap,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Audio LED Config'),
        actions: [
          IconButton(
            icon: const Icon(Icons.save),
            tooltip: 'Save Configuration',
            onPressed: _save,
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Active Scene Quick Selector
          if (scenes.isNotEmpty) ...[
            const SectionHeader('Active Scene', color: Colors.greenAccent),
            Card(
              color: const Color(0xFF1E1E1E),
              margin: const EdgeInsets.only(bottom: 20),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(8),
                side: BorderSide(color: Colors.greenAccent.withValues(alpha: 0.4)),
              ),
              child: Padding(
                padding: const EdgeInsets.all(14),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        const Text(
                          'Current Output Scene',
                          style: TextStyle(
                            fontSize: 13,
                            color: Colors.grey,
                          ),
                        ),
                        TextButton.icon(
                          style: TextButton.styleFrom(
                            foregroundColor: Colors.greenAccent,
                            padding: EdgeInsets.zero,
                            visualDensity: VisualDensity.compact,
                          ),
                          icon: const Icon(Icons.tune, size: 16),
                          label: const Text('Manage Scenes'),
                          onPressed: _openScenesManager,
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    DropdownButtonFormField<String>(
                      initialValue: scenes.any((s) => s.name == activeScene)
                          ? activeScene
                          : scenes.first.name,
                      decoration: const InputDecoration(
                        border: OutlineInputBorder(),
                        contentPadding: EdgeInsets.symmetric(
                          horizontal: 12,
                          vertical: 10,
                        ),
                      ),
                      items: scenes.map((s) {
                        return DropdownMenuItem(
                          value: s.name,
                          child: Row(
                            children: [
                              const Icon(Icons.layers, size: 18, color: Colors.greenAccent),
                              const SizedBox(width: 8),
                              Text(s.name, style: const TextStyle(fontWeight: FontWeight.bold)),
                              const SizedBox(width: 8),
                              Text(
                                '(${s.segments.length} segment${s.segments.length == 1 ? '' : 's'})',
                                style: const TextStyle(color: Colors.grey, fontSize: 12),
                              ),
                            ],
                          ),
                        );
                      }).toList(),
                      onChanged: (val) {
                        if (val != null) setState(() => activeScene = val);
                      },
                    ),
                  ],
                ),
              ),
            ),
          ],

          const SectionHeader('Sub-Pages & Customization', color: Colors.greenAccent),
          _buildNavCard(
            icon: Icons.layers_outlined,
            title: 'Scenes & Segments Manager',
            subtitle: '${scenes.length} scene${scenes.length == 1 ? '' : 's'} configured \u2022 Active: "$activeScene"',
            onTap: _openScenesManager,
          ),
          _buildNavCard(
            icon: Icons.equalizer,
            title: 'VU Meter Effect',
            subtitle: 'Gradients, threshold steps, peak hold & decay',
            trailingExtra: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                _buildColorSwatch(vu.ledLow),
                _buildColorSwatch(vu.ledMid),
                _buildColorSwatch(vu.ledHigh),
              ],
            ),
            onTap: _openVUEffectEditor,
          ),
          _buildNavCard(
            icon: Icons.bar_chart,
            title: 'Spectrum Analyzer Effect',
            subtitle: '16-band gradient interpolation colors',
            trailingExtra: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                _buildColorSwatch(spectrum.ledLow),
                _buildColorSwatch(spectrum.ledMid),
                _buildColorSwatch(spectrum.ledHigh),
              ],
            ),
            onTap: _openSpectrumEffectEditor,
          ),
          _buildNavCard(
            icon: Icons.speaker_group,
            title: 'Squeezebox / LMS Server',
            subtitle: squeezeboxConfig.server.isEmpty
                ? 'UDP Broadcast Auto-Discovery \u2022 ${squeezeboxConfig.playerName.isEmpty ? "GoLEDs VU" : squeezeboxConfig.playerName}'
                : '${squeezeboxConfig.server} \u2022 ${squeezeboxConfig.playerName.isEmpty ? "GoLEDs VU" : squeezeboxConfig.playerName}',
            onTap: _openSqueezeboxEditor,
          ),

          const SizedBox(height: 16),
          const SectionHeader('Timing & Sensitivity', color: Colors.greenAccent),
          ConfigSlider(
            label: 'LED Update Frequency',
            value: updateFreqMs.toDouble(),
            min: 10,
            max: 100,
            unit: 'ms',
            onChanged: (v) => setState(() => updateFreqMs = v.toInt()),
            activeColor: Colors.greenAccent,
          ),
          const SizedBox(height: 16),
          DbRangeSelector(
            label: 'Sensitivity Range (dB)',
            minDb: minDB,
            maxDb: maxDB,
            onChanged: (min, max) => setState(() {
              minDB = min;
              maxDB = max;
            }),
          ),
        ],
      ),
    );
  }
}
