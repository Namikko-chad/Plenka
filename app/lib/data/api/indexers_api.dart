import 'package:dio/dio.dart';

import '../models/server_settings.dart';
import '../server/server.dart';

/// Server trackers (/api/indexers) and their definitions (/api/definitions).
class IndexersApi {
  final ServerClient _server;
  IndexersApi(this._server);

  Future<List<Indexer>> list() async {
    final r = await _server.dio.get<List<dynamic>>('/api/indexers');
    return [for (final j in r.data ?? const []) Indexer.fromJson(j as Map<String, dynamic>)];
  }

  Future<IndexerDetail> get(String id) async {
    final r = await _server.dio.get<Map<String, dynamic>>('/api/indexers/${Uri.encodeComponent(id)}');
    return IndexerDetail.fromJson(r.data!);
  }

  /// A password sent empty is left unchanged by the server.
  Future<IndexerDetail> save(String id, {bool? enabled, Map<String, Object?>? config}) async {
    final r = await _server.dio.put<Map<String, dynamic>>(
      '/api/indexers/${Uri.encodeComponent(id)}',
      data: {'enabled': ?enabled, 'config': ?config},
    );
    return IndexerDetail.fromJson(r.data!);
  }

  Future<IndexerTestResult> test(String id) async {
    final r = await _server.dio.post<Map<String, dynamic>>(
      '/api/indexers/${Uri.encodeComponent(id)}/test',
      // a test is a search with site login: it can take up to the search timeout
      options: Options(receiveTimeout: const Duration(seconds: 90)),
    );
    return IndexerTestResult.fromJson(r.data);
  }

  Future<DefinitionsStatus> definitions() async {
    final r = await _server.dio.get<Map<String, dynamic>>('/api/definitions');
    return DefinitionsStatus.fromJson(r.data);
  }

  /// Re-downloads the definitions and waits for the download to finish.
  Future<DefinitionsStatus> updateDefinitions() async {
    final r = await _server.dio.post<Map<String, dynamic>>(
      '/api/definitions/update',
      options: Options(receiveTimeout: const Duration(minutes: 3)),
    );
    return DefinitionsStatus.fromJson(r.data);
  }
}
