import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../../core/utils/format.dart';
import '../../data/api/torrent_api.dart';
import '../../data/server/server.dart';
import '../../data/models/torrent.dart';
import '../../data/storage/history_store.dart';
import '../common/widgets/common.dart';
import '../player/playback.dart';
import '../router/app_router.dart';
import '../../l10n/l10n.dart';

/// Adds the release to the server's torrent engine, waits for the file list and lets you pick an
/// episode. With a single video file (a movie) or when resuming, opens the player right away
/// (except in [FilesArgs.browse] mode: there it's just a list of all files of the release).
/// Every video has a preload — the server caches the start and end of the file in advance.
/// The header button downloads the whole release and keeps it in the server's memory.
class FilesScreen extends StatefulWidget {
  final FilesArgs args;
  const FilesScreen({super.key, required this.args});

  @override
  State<FilesScreen> createState() => _FilesScreenState();
}

class _FilesScreenState extends State<FilesScreen> {
  final _ts = getIt<TorrentApi>();
  String _stage = tr.filesConnectingPeers;
  String? _hash;

  /// What the list shows (in browse mode — all files of the release).
  List<TorrentFile>? _shown;

  /// What can be played; in order — the playlist for the next episode.
  List<TorrentFile> _videos = const [];
  List<TorrentFile> _all = const [];
  Object? _error;

  /// Latest torrent state — for the preload progress line.
  Torrent? _stat;
  Timer? _pollTimer;
  DateTime? _pollUntil;
  bool _fullBusy = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _error = null;
      _shown = null;
      _stage = tr.filesConnectingPeers;
    });
    try {
      final a = widget.args;
      final poster = getIt<ServerClient>().image(a.media?.posterPath);
      final title = a.media?.title ?? a.title;
      // from history — add by hash brings the torrent back if the engine forgot it
      final added = await _ts.add(link: a.hash ?? a.torrent!.link, title: title, poster: poster, media: a.media);
      final hash = a.hash ?? added.hash;
      if (hash.isEmpty) throw Exception(tr.filesAddFailed);
      if (!mounted) return;
      setState(() => _stage = tr.filesFetchingList);
      final st = await _ts.waitForFiles(hash);
      final all = [...st.files]..sort((x, y) => naturalCompare(x.path, y.path));
      var videos = all.where((f) => f.isVideo).toList();
      if (videos.isEmpty) videos = all;
      if (!mounted) return;
      setState(() {
        _hash = hash;
        _all = all;
        _videos = videos;
        _shown = a.browse ? all : videos;
        _stat = st;
      });
      if (st.stat == 2 || _fullLoading(st)) _watchPreload();
      if (!a.browse) _autoplay(videos);
    } catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  void _autoplay(List<TorrentFile> files) {
    final resumeId = widget.args.resumeFileId;
    final resume = resumeId == null ? null : files.where((f) => f.id == resumeId).firstOrNull;
    final target = resume ?? (files.length == 1 ? files.first : null);
    if (target != null) _play(target, replace: true);
  }

  void _play(TorrentFile f, {bool replace = false}) {
    startPlayback(
      context,
      PlayerArgs(
        hash: _hash!,
        file: f,
        playlist: _videos,
        files: _all,
        torrentTitle: widget.args.title,
        media: widget.args.media,
      ),
      replace: replace,
    );
  }

  Future<void> _preload(TorrentFile f) async {
    final messenger = ScaffoldMessenger.of(context);
    final l = context.l10n;
    try {
      await _ts.preload(_hash!, f.id);
      messenger.showSnackBar(SnackBar(content: Text(l.preloadStarted(f.name))));
      _watchPreload();
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(errorText(e))));
    }
  }

  static bool _fullLoading(Torrent st) => st.fullDownload && st.loadedSize < st.torrentSize;

  Future<void> _toggleFullDownload() async {
    final st = _stat;
    if (_fullBusy || _hash == null) return;
    final messenger = ScaffoldMessenger.of(context);
    final l = context.l10n;
    setState(() => _fullBusy = true);
    try {
      if (st?.fullDownload == true) {
        await _ts.cancelDownload(_hash!);
        final next = await _ts.get(_hash!);
        if (mounted) setState(() => _stat = next);
        messenger.showSnackBar(SnackBar(content: Text(l.downloadAllCancelled)));
      } else {
        final next = await _ts.downloadAll(_hash!);
        if (mounted) setState(() => _stat = next);
        messenger.showSnackBar(SnackBar(content: Text(l.downloadAllStarted)));
        _watchPreload();
      }
    } on DioException catch (e) {
      final noRoom = e.response?.statusCode == 507;
      messenger.showSnackBar(SnackBar(content: Text(noRoom ? l.downloadAllNoRoom : errorText(e))));
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(errorText(e))));
    } finally {
      if (mounted) setState(() => _fullBusy = false);
    }
  }

  /// Polls the torrent while preloading (stat 2) or downloading in full; for the first seconds
  /// regardless, until the server moves the torrent into that status.
  void _watchPreload() {
    _pollUntil = DateTime.now().add(const Duration(seconds: 10));
    _pollTimer ??= Timer.periodic(const Duration(seconds: 2), (_) => _poll());
  }

  Future<void> _poll() async {
    try {
      final st = await _ts.get(_hash!);
      if (!mounted) return;
      setState(() => _stat = st);
      if (st.stat != 2 && !_fullLoading(st) && DateTime.now().isAfter(_pollUntil!)) {
        _pollTimer?.cancel();
        _pollTimer = null;
      }
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    final shown = _shown;
    final l = context.l10n;
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.args.media?.title ?? l.filesTitle),
        actions: [
          if (_hash != null)
            _fullBusy
                ? const Padding(
                    padding: EdgeInsets.symmetric(horizontal: 14),
                    child: SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)),
                  )
                : IconButton(
                    tooltip: _stat?.fullDownload == true ? l.downloadAllCancel : l.downloadAll,
                    onPressed: _toggleFullDownload,
                    icon: Icon(
                      _stat?.fullDownload == true ? Icons.cancel_outlined : Icons.download_for_offline_outlined,
                    ),
                  ),
        ],
      ),
      body: PageBackground(
        child: Builder(
          builder: (context) {
            if (_error != null) return MessageView.error(_error!, onRetry: _load);
            if (shown == null) return LoadingIndicator(message: '$_stage\n${widget.args.title}');
            final history = getIt<HistoryStore>();
            final playable = _videos.toSet();
            // focus the last watched file so a remote can resume with one press
            final entries = {for (final f in shown) f.id: history.entry(_hash!, f.id)};
            final last = entries.values.whereType<WatchEntry>().fold<WatchEntry?>(
              null,
              (best, e) => best == null || e.updatedAt.isAfter(best.updatedAt) ? e : best,
            );
            final firstPlayable = shown.where(playable.contains).firstOrNull;
            final st = _stat;
            final preloading = st != null && st.stat == 2 && st.preloadSize > 0;
            return ListView.builder(
              padding: pagePadding(context).copyWith(bottom: 24),
              itemCount: shown.length + 1,
              itemBuilder: (_, i) {
                if (i == 0) {
                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      SectionTitle(widget.args.title, inset: false),
                      if (st != null && st.fullDownload && st.torrentSize > 0)
                        Padding(
                          padding: const EdgeInsets.only(bottom: 12),
                          child: _FullProgress(torrent: st),
                        ),
                      if (preloading)
                        Padding(
                          padding: const EdgeInsets.only(bottom: 12),
                          child: _PreloadProgress(torrent: st),
                        ),
                    ],
                  );
                }
                final f = shown[i - 1];
                final canPlay = playable.contains(f);
                final focus = last == null ? f == firstPlayable : last.fileId == f.id;
                return Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: canPlay
                      ? Row(
                          children: [
                            Expanded(
                              child: FocusCard(
                                autofocus: focus,
                                focusScale: 1.01,
                                onTap: () => _play(f),
                                child: _FileRow(file: f, entry: entries[f.id], playable: true),
                              ),
                            ),
                            const SizedBox(width: 8),
                            IconButton(
                              tooltip: l.preload,
                              onPressed: () => _preload(f),
                              icon: const Icon(Icons.downloading),
                            ),
                          ],
                        )
                      : Opacity(
                          opacity: 0.5,
                          child: AppCard(child: _FileRow(file: f, entry: null, playable: false)),
                        ),
                );
              },
            );
          },
        ),
      ),
    );
  }
}

/// "Preload: 120 MB / 400 MB" with a bar.
class _PreloadProgress extends StatelessWidget {
  final Torrent torrent;
  const _PreloadProgress({required this.torrent});

  @override
  Widget build(BuildContext context) {
    final t = torrent;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          context.l10n.preloadProgress(formatBytes(t.preloadedBytes), formatBytes(t.preloadSize)),
          style: AppText.small.copyWith(color: AppColors.accent),
        ),
        const SizedBox(height: 6),
        ProgressLine(value: (t.preloadedBytes / t.preloadSize).clamp(0, 1).toDouble()),
      ],
    );
  }
}

/// "In full: 1.2 GB / 4 GB" with a bar; when done — "Downloaded in full".
class _FullProgress extends StatelessWidget {
  final Torrent torrent;
  const _FullProgress({required this.torrent});

  @override
  Widget build(BuildContext context) {
    final t = torrent;
    final done = t.loadedSize >= t.torrentSize;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          done
              ? context.l10n.downloadAllDone(formatBytes(t.torrentSize))
              : context.l10n.downloadAllProgress(formatBytes(t.loadedSize), formatBytes(t.torrentSize)),
          style: AppText.small.copyWith(color: done ? AppColors.ok : AppColors.accent),
        ),
        if (!done) ...[
          const SizedBox(height: 6),
          ProgressLine(value: (t.loadedSize / t.torrentSize).clamp(0, 1).toDouble()),
        ],
      ],
    );
  }
}

class _FileRow extends StatelessWidget {
  final TorrentFile file;
  final WatchEntry? entry;
  final bool playable;
  const _FileRow({required this.file, required this.entry, required this.playable});

  @override
  Widget build(BuildContext context) {
    final e = entry;
    final f = file;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Row(
        children: [
          Icon(
            !playable
                ? Icons.insert_drive_file_outlined
                : (e?.finished == true ? Icons.check_circle : Icons.play_circle_outline),
            color: e?.finished == true ? AppColors.ok : AppColors.muted,
            size: 18,
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(f.name, maxLines: 2, overflow: TextOverflow.ellipsis, style: AppText.body),
                if (e != null && !e.finished && e.progress > 0) ...[
                  const SizedBox(height: 8),
                  ProgressLine(value: e.progress),
                ],
              ],
            ),
          ),
          const SizedBox(width: 12),
          Text(formatBytes(f.length), style: AppText.small),
        ],
      ),
    );
  }
}
