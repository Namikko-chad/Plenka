import '../models/server_settings.dart';
import '../server/server.dart';

/// Server settings: catalog and search (/api/config).
class ConfigApi {
  final ServerClient _server;
  ConfigApi(this._server);

  Future<ServerConfig> get() async {
    final r = await _server.dio.get<Map<String, dynamic>>('/api/config');
    return ServerConfig.fromJson(r.data);
  }

  /// Any subset of fields. `tmdbApiKey: ""` keeps the key, `"-"` removes it.
  Future<ServerConfig> put(Map<String, Object?> fields) async {
    final r = await _server.dio.put<Map<String, dynamic>>('/api/config', data: fields);
    return ServerConfig.fromJson(r.data);
  }
}
