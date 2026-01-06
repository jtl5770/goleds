import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:provider/provider.dart';
import '../providers/config_provider.dart';
import '../widgets/producer_card.dart';
import 'editors/sensor_led_editor.dart';
import 'editors/cylon_led_editor.dart';
import 'editors/night_led_editor.dart';
import 'editors/clock_led_editor.dart';
import 'editors/audio_led_editor.dart';
import 'editors/multi_blob_led_editor.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  @override
  void initState() {
    super.initState();
  }

  void _showSettingsDialog() {
    final provider = context.read<ConfigProvider>();
    final controller = TextEditingController(text: provider.baseUrl);

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Connect to Go-LEDS'),
        content: TextField(
          controller: controller,
          decoration: const InputDecoration(
            labelText: 'Server URL',
            hintText: 'http://192.168.1.x:8080',
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
              provider.setBaseUrl(controller.text);
              Navigator.pop(ctx);
            },
            child: const Text('Connect'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final configProvider = context.watch<ConfigProvider>();
    final config = configProvider.config;
    final isLoading = configProvider.isLoading;
    final error = configProvider.error;

    return Scaffold(
      appBar: AppBar(
        title: const Text('GO-LEDS COMMANDER'),
        actions: [
          if (!kIsWeb)
            IconButton(
              icon: const Icon(Icons.settings),
              onPressed: _showSettingsDialog,
            ),
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () => configProvider.fetchConfig(),
          ),
        ],
      ),
      body: isLoading && config == null
          ? const Center(child: CircularProgressIndicator())
          : error != null
              ? Center(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      const Icon(Icons.error_outline, size: 48, color: Colors.redAccent),
                      const SizedBox(height: 16),
                      Text('Connection Failed', style: Theme.of(context).textTheme.headlineSmall),
                      Padding(
                        padding: const EdgeInsets.all(16.0),
                        child: Text(error, textAlign: TextAlign.center),
                      ),
                      if (!kIsWeb)
                        ElevatedButton(
                          onPressed: _showSettingsDialog,
                          child: const Text('Check Settings'),
                        )
                    ],
                  ),
                )
              : config == null
                  ? const Center(child: Text('No Configuration Loaded'))
                  : Padding(
                      padding: const EdgeInsets.all(16.0),
                      child: GridView.count(
                        crossAxisCount: MediaQuery.of(context).size.width > 600 ? 2 : 1,
                        crossAxisSpacing: 16,
                        mainAxisSpacing: 12,
                        childAspectRatio: 3.5,
                        children: [
                          ProducerCard(
                            title: 'Sensor',
                            icon: Icons.sensors,
                            isEnabled: config.sensorLED.enabled,
                            accentColor: Colors.purpleAccent,
                            onToggle: () => configProvider.toggleProducer('SensorLED', !config.sensorLED.enabled),
                            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const SensorLEDEditor())),
                          ),
                          ProducerCard(
                            title: 'Night Light',
                            icon: Icons.nightlight_round,
                            isEnabled: config.nightLED.enabled,
                            accentColor: Colors.orangeAccent,
                            onToggle: () => configProvider.toggleProducer('NightLED', !config.nightLED.enabled),
                            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const NightLEDEditor())),
                          ),
                          ProducerCard(
                            title: 'Clock',
                            icon: Icons.access_time,
                            isEnabled: config.clockLED.enabled,
                            accentColor: Colors.blueAccent,
                            onToggle: () => configProvider.toggleProducer('ClockLED', !config.clockLED.enabled),
                            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const ClockLEDEditor())),
                          ),
                          ProducerCard(
                            title: 'Audio VU',
                            icon: Icons.equalizer,
                            isEnabled: config.audioLED.enabled,
                            accentColor: Colors.greenAccent,
                            onToggle: () => configProvider.toggleProducer('AudioLED', !config.audioLED.enabled),
                            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const AudioLEDEditor())),
                          ),
                          ProducerCard(
                            title: 'Cylon Eye',
                            icon: Icons.remove_red_eye,
                            isEnabled: config.cylonLED.enabled,
                            isDisabled: !config.sensorLED.enabled,
                            accentColor: Colors.redAccent,
                            onToggle: () => configProvider.toggleProducer('CylonLED', !config.cylonLED.enabled),
                            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const CylonLEDEditor())),
                          ),
                          ProducerCard(
                            title: 'Multi Blob',
                            icon: Icons.bubble_chart,
                            isEnabled: config.multiBlobLED.enabled,
                            isDisabled: !config.sensorLED.enabled,
                            accentColor: Colors.pinkAccent,
                            onToggle: () => configProvider.toggleProducer('MultiBlobLED', !config.multiBlobLED.enabled),
                            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const MultiBlobLEDEditor())),
                          ),
                        ],
                      ),
                    ),
    );
  }
}

