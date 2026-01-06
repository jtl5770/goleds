import 'dart:convert';
import 'package:http/http.dart' as http;
import 'models.dart';

class ApiService {
  String baseUrl;

  ApiService(this.baseUrl);

  void updateUrl(String url) {
    if (url.endsWith('/')) {
      baseUrl = url.substring(0, url.length - 1);
    } else {
      baseUrl = url;
    }
  }

  Future<RuntimeConfig> fetchConfig() async {
    final uri = Uri.parse('$baseUrl/api/config');
    try {
      final response = await http.get(uri).timeout(const Duration(seconds: 5));

      if (response.statusCode == 200) {
        return RuntimeConfig.fromJson(jsonDecode(response.body));
      } else {
        throw Exception(
          'Server returned ${response.statusCode}\nBody: ${response.body}',
        );
      }
    } on http.ClientException catch (e) {
      throw Exception(
        'HTTP Client Error: ${e.message}\nCheck if the server is running at $uri',
      );
    } catch (e) {
      rethrow;
    }
  }

  Future<void> saveConfig(RuntimeConfig config) async {
    final uri = Uri.parse('$baseUrl/api/config');
    try {
      final response = await http
          .post(
            uri,
            headers: <String, String>{
              'Content-Type': 'application/json; charset=UTF-8',
            },
            body: jsonEncode(config.toJson()),
          )
          .timeout(const Duration(seconds: 5));

      if (response.statusCode != 200) {
        throw Exception(
          'Failed to save (Status: ${response.statusCode})\nError: ${response.body}',
        );
      }
    } catch (e) {
      rethrow;
    }
  }
}
