import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models.dart';
import '../api_service.dart';

class ConfigProvider with ChangeNotifier {
  final ApiService _apiService;
  RuntimeConfig? _config;
  bool _isLoading = false;
  String? _error;
  static const String _urlKey = 'go_leds_base_url';
  static const String _defaultUrl = 'http://goleds.local:8080';

  ConfigProvider() : _apiService = ApiService(_defaultUrl) {
    _init();
  }

  RuntimeConfig? get config => _config;
  bool get isLoading => _isLoading;
  String? get error => _error;
  String get baseUrl => _apiService.baseUrl;

  Future<void> _init() async {
    final prefs = await SharedPreferences.getInstance();
    final savedUrl = prefs.getString(_urlKey);
    if (savedUrl != null) {
      _apiService.updateUrl(savedUrl);
    }
    fetchConfig();
  }

  Future<void> setBaseUrl(String url) async {
    _apiService.updateUrl(url);
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_urlKey, url);
    fetchConfig();
  }

  Future<void> fetchConfig({int retries = 9}) async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    int attempt = 0;
    while (attempt <= retries) {
      try {
        _config = await _apiService.fetchConfig();
        _error = null;
        break; // Success
      } catch (e) {
        attempt++;
        final msg = e.toString();
        if (attempt <= retries &&
            (msg.contains("Invalid response line") ||
                msg.contains("Connection closed") ||
                msg.contains("Connection refused"))) {
          // Wait a bit for server to come back up
          await Future.delayed(Duration(milliseconds: 500 * attempt));
          continue;
        }
        _error = "Attempt $attempt failed:\n$msg";
      }
    }

    _isLoading = false;
    notifyListeners();
  }

  Future<void> updateConfig(RuntimeConfig newConfig) async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      await _apiService.saveConfig(newConfig);
      _config = newConfig;
      // After a successful save, the server will restart.
      // We give it a moment then fetch the latest state to confirm.
      await Future.delayed(const Duration(milliseconds: 500));
      await fetchConfig();
    } catch (e) {
      final msg = e.toString();
      if (msg.contains("Invalid response line") ||
          msg.contains("Connection closed")) {
        // Server likely restarted immediately after save. Treat as success and reload.
        await fetchConfig();
        return;
      }
      _error = "Save failed: $msg";
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  void toggleProducer(String producerName, bool isEnabled) {
    if (_config == null) return;

    switch (producerName) {
      case 'SensorLED':
        _config!.sensorLED.enabled = isEnabled;
        break;
      case 'NightLED':
        _config!.nightLED.enabled = isEnabled;
        break;
      case 'ClockLED':
        _config!.clockLED.enabled = isEnabled;
        break;
      case 'AudioLED':
        _config!.audioLED.enabled = isEnabled;
        break;
      case 'CylonLED':
        _config!.cylonLED.enabled = isEnabled;
        break;
      case 'MultiBlobLED':
        _config!.multiBlobLED.enabled = isEnabled;
        break;
    }

    updateConfig(_config!);
  }
}
