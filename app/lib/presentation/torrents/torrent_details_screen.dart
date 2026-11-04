import 'dart:async';
import 'dart:math' as math;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../../core/utils/format.dart';
import '../../data/api/torrent_api.dart';
import '../../data/models/torrent.dart';
import '../../data/storage/history_store.dart';
import '../../l10n/l10n.dart';
import '../common/widgets/common.dart';
import 'torrents_cubit.dart';
import 'torrents_screen.dart';

/// One engine torrent: stats, cache map and actions (preload, full download, files, delete).
class TorrentDetailsScreen extends StatefulWidget {
  final Torrent torrent;
  const TorrentDetailsScreen({super.key, required this.torrent});

  @override
  State<TorrentDetailsScreen> createState() => _TorrentDetailsScreenState();
}

class _TorrentDetailsScreenState extends State<TorrentDetailsScreen> {
  final _api = getIt<TorrentApi>();
  late Torrent _t = widget.torrent;
  CacheState? _cache;
  Timer? _timer;
  bool _preloading = false;
  bool _fullBusy = false;

  @override
  void initState() {
    super.initState();
    _poll();
    _timer = Timer.periodic(const Duration(seconds: 2), (_) => _poll());
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  Future<void> _poll() async {
    try {
      final r = await Future.wait([_api.get(_t.hash), _api.cache(_t.hash)]);
      if (!mounted) return;
      setState(() {
        _t = r[0] as Torrent;
        _cache = r[1] as CacheState;
      });
    } catch (_) {
      // a missed tick: the next one retries
    }
  }

  void _snack(String text) => ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(text)));

  Future<void> _preload(TorrentFile f) async {
    if (_preloading) return;
    setState(() => _preloading = true);
    try {
      await _api.preload(_t.hash, f.id);
      if (mounted) _snack(context.l10n.preloadStarted(f.name));
    } catch (e) {
      if (mounted) _snack(errorText(e));
    } finally {
      if (mounted) setState(() => _preloading = false);
    }
  }

  Future<void> _toggleFull() async {
    if (_fullBusy) return;
    final l = context.l10n;
    setState(() => _fullBusy = true);
    try {
      if (_t.fullDownload) {
        await _api.cancelDownload(_t.hash);
        if (mounted) setState(() => _t = _t.copyWith(fullDownload: false));
        _snack(l.downloadAllCancelled);
      } else {
        final next = await _api.downloadAll(_t.hash);
        if (mounted) setState(() => _t = next);
        _snack(l.downloadAllStarted);
      }
    } on DioException catch (e) {
      if (mounted) _snack(e.response?.statusCode == 507 ? l.downloadAllNoRoom : errorText(e));
    } catch (e) {
      if (mounted) _snack(errorText(e));
    } finally {
      if (mounted) setState(() => _fullBusy = false);
    }
  }

  Future<void> _remove() async {
    if (await removeTorrent(context, _api.remove, _t) && mounted) Navigator.of(context).pop();
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final t = _t;
    final target = preloadTarget(t, getIt<HistoryStore>().entries);
    final rows = <(String, String)>[
      (l.torrentsStatus, t.statString.isEmpty ? '—' : t.statString),
      (l.torrentsSize, formatBytes(t.torrentSize)),
      (l.torrentsLoaded, formatBytes(t.loadedSize > 0 ? t.loadedSize : t.bytesRead)),
      (l.torrentsSpeedDown, formatSpeed(t.downloadSpeed)),
      (l.torrentsSpeedUp, formatSpeed(t.uploadSpeed)),
      (l.torrentsPeers, '${t.activePeers} / ${t.totalPeers}'),
      (l.torrentsSeeders, '${t.connectedSeeders}'),
      if (t.fullDownload && t.torrentSize > 0)
        (
          l.torrentsFull,
          '${formatBytes(t.loadedSize)} / ${formatBytes(t.torrentSize)} · ${((t.fullProgress ?? 0) * 100).floor()}%',
        ),
      if (t.preloadSize > 0) (l.preload, '${formatBytes(t.preloadedBytes)} / ${formatBytes(t.preloadSize)}'),
      if (DateTime.tryParse(t.addedAt) case final added?) (l.torrentsAddedAt, formatDateTime(added)),
    ];
    final cache = _cache;
    return Scaffold(
      appBar: AppBar(title: Text(t.displayTitle, maxLines: 1, overflow: TextOverflow.ellipsis)),
      body: PageBackground(
        child: ListView(
          padding: pagePadding(context).copyWith(top: 20, bottom: 40),
          children: [
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                OutlinedButton.icon(
                  autofocus: true,
                  onPressed: () => openTorrentFiles(context, t),
                  icon: const Icon(Icons.folder_open_outlined, size: 16),
                  label: Text(l.filesTitle),
                ),
                if (target != null)
                  OutlinedButton.icon(
                    onPressed: _preloading || t.stat == 2 ? null : () => _preload(target),
                    icon: _preloading || t.stat == 2 ? const Spinner() : const Icon(Icons.downloading, size: 16),
                    label: Text(l.preload),
                  ),
                OutlinedButton.icon(
                  style: t.fullDownload ? OutlinedButton.styleFrom(foregroundColor: AppColors.error) : null,
                  onPressed: _fullBusy ? null : _toggleFull,
                  icon: _fullBusy
                      ? const Spinner()
                      : Icon(t.fullDownload ? Icons.close : Icons.download_outlined, size: 16),
                  label: Text(t.fullDownload ? l.downloadAllCancel : l.downloadAll),
                ),
                OutlinedButton.icon(
                  style: OutlinedButton.styleFrom(foregroundColor: AppColors.error),
                  onPressed: _remove,
                  icon: const Icon(Icons.delete_outline, size: 16),
                  label: Text(l.torrentsDelete),
                ),
              ],
            ),
            SectionTitle(l.torrentsDetails, inset: false),
            AppCard(
              padding: const EdgeInsets.all(16),
              child: Column(
                children: [
                  for (final (k, v) in rows) _Row(k, v),
                  _Row(
                    l.torrentsHash,
                    t.hash,
                    onTap: () {
                      Clipboard.setData(ClipboardData(text: t.hash));
                      _snack(l.torrentsHashCopied);
                    },
                  ),
                ],
              ),
            ),
            SectionTitle(l.torrentsCache, inset: false),
            if (cache == null)
              const LoadingIndicator()
            else if (cache.piecesCount == 0)
              Text(l.torrentsNoCache, style: AppText.small)
            else ...[
              Text(
                l.torrentsCacheSummary(
                  formatBytes(cache.filled),
                  formatBytes(cache.capacity),
                  cache.piecesCount,
                  formatBytes(cache.piecesLength),
                ),
                style: AppText.small,
              ),
              const SizedBox(height: 12),
              PieceMap(cache: cache),
              const SizedBox(height: 12),
              Wrap(
                spacing: 14,
                runSpacing: 6,
                children: [
                  _Legend(color: AppColors.accent, label: l.torrentsLegendDone),
                  _Legend(color: AppColors.accent.withValues(alpha: 0.45), label: l.torrentsLegendPartial),
                  _Legend(color: AppColors.warning, label: l.torrentsLegendQueued),
                  _Legend(color: AppColors.purpleLight, label: l.torrentsLegendReader, outline: true),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _Row extends StatelessWidget {
  final String k;
  final String v;
  final VoidCallback? onTap;
  const _Row(this.k, this.v, {this.onTap});

  @override
  Widget build(BuildContext context) {
    final value = Text(
      v,
      textAlign: TextAlign.right,
      style: AppText.small.copyWith(color: onTap != null ? AppColors.accent : AppColors.text),
    );
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(k, style: AppText.small),
          const SizedBox(width: 16),
          Expanded(
            child: onTap == null
                ? value
                : InkWell(onTap: onTap, borderRadius: BorderRadius.circular(AppRadius.badge), child: value),
          ),
        ],
      ),
    );
  }
}

class _Legend extends StatelessWidget {
  final Color color;
  final String label;
  final bool outline;
  const _Legend({required this.color, required this.label, this.outline = false});

  @override
  Widget build(BuildContext context) => Row(
    mainAxisSize: MainAxisSize.min,
    children: [
      Container(
        width: 10,
        height: 10,
        decoration: BoxDecoration(
          color: outline ? null : color,
          border: outline ? Border.all(color: color) : null,
          borderRadius: BorderRadius.circular(2),
        ),
      ),
      const SizedBox(width: 6),
      Text(label, style: AppText.small),
    ],
  );
}

/// Compact cache map: one cell per bucket of pieces, at most 24 rows.
class PieceMap extends StatelessWidget {
  final CacheState cache;
  const PieceMap({super.key, required this.cache});

  static const double cell = 6;
  static const double gap = 1;
  static const int maxRows = 24;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, c) {
        final layout = PieceMapLayout.of(cache, c.maxWidth);
        return CustomPaint(size: Size(c.maxWidth, layout.rows * (cell + gap)), painter: _PieceMapPainter(layout));
      },
    );
  }
}

/// Cells of the cache map: fill fraction, queued and being-read flags per cell.
class PieceMapLayout {
  final int cols;
  final int rows;
  final List<double> fill;
  final List<bool> queued;
  final List<bool> reading;

  const PieceMapLayout(this.cols, this.rows, this.fill, this.queued, this.reading);

  factory PieceMapLayout.of(CacheState c, double width) {
    final count = math.max(0, c.piecesCount);
    final cols = math.max(1, ((width + PieceMap.gap) / (PieceMap.cell + PieceMap.gap)).floor());
    final cells = math.min(count, cols * PieceMap.maxRows);
    final per = cells > 0 ? count / cells : 1.0;
    final fill = List<double>.filled(cells, 0);
    final queued = List<bool>.filled(cells, false);
    final reading = List<bool>.filled(cells, false);
    if (cells > 0) {
      for (final p in c.pieces) {
        final i = math.min(cells - 1, (p.id / per).floor());
        if (i < 0) continue;
        fill[i] += p.fill / per;
        if (p.priority > 0 && !p.completed) queued[i] = true;
      }
      final step = math.max(1, per.floor());
      for (final (start, end) in c.readers) {
        for (var id = start; id <= end && id < count; id += step) {
          reading[math.min(cells - 1, (id / per).floor())] = true;
        }
      }
    }
    return PieceMapLayout(cols, math.max(1, (cells / cols).ceil()), fill, queued, reading);
  }
}

class _PieceMapPainter extends CustomPainter {
  final PieceMapLayout m;
  const _PieceMapPainter(this.m);

  @override
  void paint(Canvas canvas, Size size) {
    const step = PieceMap.cell + PieceMap.gap;
    final paint = Paint();
    final stroke = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1
      ..color = AppColors.purpleLight;
    for (var i = 0; i < m.fill.length; i++) {
      final rect = Rect.fromLTWH((i % m.cols) * step, (i ~/ m.cols) * step, PieceMap.cell, PieceMap.cell);
      canvas.drawRect(rect, paint..color = AppColors.border);
      final f = math.min(1.0, m.fill[i]);
      if (f > 0) {
        canvas.drawRect(rect, paint..color = AppColors.accent.withValues(alpha: f >= 0.999 ? 1 : 0.25 + 0.6 * f));
      } else if (m.queued[i]) {
        canvas.drawRect(rect, paint..color = AppColors.warning.withValues(alpha: 0.7));
      }
      if (m.reading[i]) canvas.drawRect(rect.deflate(0.5), stroke);
    }
  }

  @override
  bool shouldRepaint(covariant _PieceMapPainter old) => old.m != m;
}
