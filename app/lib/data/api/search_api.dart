import 'package:dio/dio.dart';

import '../models/torrent.dart';
import '../server/server.dart';

/// How one tracker responded (an `indexers` item in the /api/search response).
class IndexerStatus {
  final String id;
  final String name;
  final bool ok;

  const IndexerStatus({required this.id, required this.name, required this.ok});

  factory IndexerStatus.fromJson(Map<String, dynamic> j) =>
      IndexerStatus(id: j['id'] as String? ?? '', name: j['name'] as String? ?? '', ok: j['status'] == 'ok');
}

/// Response to a single /api/search request.
class SearchResponse {
  final List<TorrentResult> results;
  final List<IndexerStatus> indexers;
  const SearchResponse(this.results, this.indexers);

  factory SearchResponse.fromJson(Map<String, dynamic>? j) => SearchResponse(
    [for (final r in (j?['results'] as List? ?? const [])) ?TorrentResult.fromJson(r as Map<String, dynamic>)],
    [for (final i in (j?['indexers'] as List? ?? const [])) IndexerStatus.fromJson(i as Map<String, dynamic>)],
  );
}

/// Combined result of searching several title variants.
class SearchOutcome {
  final List<TorrentResult> results;

  /// One status per tracker: "ok" if it answered at least one request.
  final List<IndexerStatus> indexers;

  const SearchOutcome(this.results, this.indexers);

  /// Merges responses: dedupe by infoHash (or title+size), sort by seeders.
  static SearchOutcome merge(Iterable<SearchResponse> responses) {
    final byKey = <String, TorrentResult>{};
    final byIndexer = <String, IndexerStatus>{};
    for (final r in responses) {
      for (final t in r.results) {
        byKey.putIfAbsent(t.dedupKey, () => t);
      }
      for (final i in r.indexers) {
        final prev = byIndexer[i.id];
        if (prev == null || (!prev.ok && i.ok)) byIndexer[i.id] = i;
      }
    }
    final results = byKey.values.toList()..sort((a, b) => b.seeders.compareTo(a.seeders));
    final indexers = byIndexer.values.toList()..sort((a, b) => a.name.compareTo(b.name));
    return SearchOutcome(results, indexers);
  }
}

/// Torrent search across the server's trackers (GET /api/search).
class SearchApi {
  final ServerClient _server;
  SearchApi(this._server);

  Future<SearchResponse> _search(String query) async {
    final r = await _server.dio.get<Map<String, dynamic>>(
      '/api/search',
      queryParameters: {'q': query},
      // the server waits for all trackers up to its own timeout (searchTimeoutSec)
      options: Options(receiveTimeout: const Duration(seconds: 90)),
    );
    return SearchResponse.fromJson(r.data);
  }

  /// Searches several title variants in parallel. Fails only if all requests fail.
  Future<SearchOutcome> search(List<String> queries) async {
    final unique = {for (final q in queries) q.trim()}..removeWhere((q) => q.isEmpty);
    if (unique.isEmpty) return const SearchOutcome([], []);
    Object? firstError;
    final responses = await Future.wait(
      unique.map((q) async {
        try {
          return await _search(q);
        } catch (e) {
          firstError ??= e;
          return null;
        }
      }),
    );
    final ok = responses.whereType<SearchResponse>().toList();
    if (ok.isEmpty) throw firstError!;
    return SearchOutcome.merge(ok);
  }
}
