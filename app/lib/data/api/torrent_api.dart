import 'dart:convert';

import 'package:dio/dio.dart';

import '../models/media.dart';
import '../models/torrent.dart';
import '../server/server.dart';
import '../../l10n/l10n.dart';

/// The server's torrent engine: /api/torrents and /api/stream streams.
class TorrentApi {
  final ServerClient _server;
  TorrentApi(this._server);

  static Torrent _parse(Object? data) =>
      Torrent.fromJson((data is String ? jsonDecode(data) : data) as Map<String, dynamic>);

  /// POST /api/torrents body. [link] — magnet, hex infohash or a search result link as is
  /// (including a relative /api/indexers/…/download). `data` holds the movie card for the web UI.
  static Map<String, dynamic> addBody({
    required String link,
    required String title,
    String? poster,
    MediaItem? media,
  }) => {'link': link, 'title': title, 'poster': ?poster, if (media != null) 'data': jsonEncode(media.toJson())};

  /// Adds a torrent (the server always saves it, so "Continue watching" survives a
  /// restart). The response is immediate; metadata comes later, see [waitForFiles].
  Future<Torrent> add({required String link, required String title, String? poster, MediaItem? media}) async {
    final r = await _server.dio.post<dynamic>(
      '/api/torrents',
      data: addBody(link: link, title: title, poster: poster, media: media),
    );
    return _parse(r.data);
  }

  /// GET /api/torrents — every saved torrent, loaded or not.
  Future<List<Torrent>> list() async {
    final r = await _server.dio.get<dynamic>('/api/torrents');
    final data = r.data is String ? jsonDecode(r.data as String) : r.data;
    return [for (final j in (data as List? ?? const [])) Torrent.fromJson(j as Map<String, dynamic>)];
  }

  /// DELETE /api/torrents/{hash} — removes the torrent and its cache.
  Future<void> remove(String hash) async {
    await _server.dio.delete<dynamic>('/api/torrents/$hash');
  }

  /// POST /api/torrents/{hash}/drop — unloads the torrent from memory; it stays in the list (stat 5).
  Future<void> drop(String hash) async {
    await _server.dio.post<dynamic>('/api/torrents/$hash/drop');
  }

  /// GET /api/torrents/{hash}/cache — the cache map.
  Future<CacheState> cache(String hash) async {
    final r = await _server.dio.get<dynamic>('/api/torrents/$hash/cache');
    return CacheState.fromJson((r.data is String ? jsonDecode(r.data as String) : r.data) as Map<String, dynamic>);
  }

  /// GET /api/torrents/{hash}; an unknown hash is 404.
  Future<Torrent> get(String hash) async {
    final r = await _server.dio.get<dynamic>('/api/torrents/$hash');
    return _parse(r.data);
  }

  /// POST /api/torrents/{hash}/preload — the server downloads the head and tail of every file to
  /// disk in the background, this one first. Waits for metadata (up to a minute), then 202;
  /// 507 — not enough disk space. Progress is visible in [get]: `stat == 2`,
  /// `preloadedBytes` / `preloadSize`.
  Future<void> preload(String hash, int fileId) async {
    await _server.dio.post<dynamic>(
      '/api/torrents/$hash/preload',
      data: {'fileId': fileId},
      options: Options(receiveTimeout: const Duration(seconds: 75)),
    );
  }

  /// POST /api/torrents/{hash}/download — the server downloads the whole torrent to disk and keeps
  /// it until cancelled ([cancelDownload]) or deleted. Waits for metadata
  /// (up to a minute); 507 — the torrent doesn't fit on the server's disk.
  Future<Torrent> downloadAll(String hash) async {
    final r = await _server.dio.post<dynamic>(
      '/api/torrents/$hash/download',
      options: Options(receiveTimeout: const Duration(seconds: 75)),
    );
    return _parse(r.data);
  }

  /// DELETE /api/torrents/{hash}/download — cancels the full download and deletes its data,
  /// except the preload.
  Future<void> cancelDownload(String hash) async {
    await _server.dio.delete<dynamic>('/api/torrents/$hash/download');
  }

  /// Waits until the engine gets the metadata (file list) from peers.
  Future<Torrent> waitForFiles(String hash, {Duration timeout = const Duration(seconds: 90)}) async {
    final deadline = DateTime.now().add(timeout);
    while (true) {
      try {
        final st = await get(hash);
        if (st.files.isNotEmpty) return st;
      } on DioException catch (e) {
        // right after add the torrent may not be listed yet
        if (e.response?.statusCode != 404) rethrow;
      }
      if (DateTime.now().isAfter(deadline)) {
        throw Exception(tr.filesNoPeers);
      }
      await Future<void>.delayed(const Duration(seconds: 1));
    }
  }

  /// `{server}/api/stream/{hash}/{file id}/{file name}` — the name is for the player; the server ignores it.
  static String buildStreamUrl(String baseUrl, String hash, TorrentFile file) =>
      '$baseUrl/api/stream/$hash/${file.id}/${Uri.encodeComponent(file.name)}';

  String streamUrl(String hash, TorrentFile file) => buildStreamUrl(_server.baseUrl, hash, file);
}
