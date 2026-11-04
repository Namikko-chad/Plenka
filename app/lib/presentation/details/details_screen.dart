import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../../data/api/tmdb_api.dart';
import '../../data/server/server.dart';
import '../../data/models/media.dart';
import '../../data/storage/history_store.dart';
import '../common/widgets/common.dart';
import '../router/app_router.dart';
import '../../l10n/l10n.dart';

class DetailsScreen extends StatefulWidget {
  final MediaItem item;
  const DetailsScreen({super.key, required this.item});

  @override
  State<DetailsScreen> createState() => _DetailsScreenState();
}

class _DetailsScreenState extends State<DetailsScreen> {
  late Future<MediaDetails> _details = getIt<TmdbApi>().details(widget.item);

  @override
  Widget build(BuildContext context) {
    final server = getIt<ServerClient>();
    final item = widget.item;
    final wide = isWide(context);
    return Scaffold(
      body: PageBackground(
        child: Stack(
          fit: StackFit.expand,
          children: [
            // Background: a still from the movie, darkened towards the bottom and the left edge.
            if (item.backdropPath != null)
              Positioned(
                top: 0,
                left: 0,
                right: 0,
                height: wide ? MediaQuery.sizeOf(context).height : 320,
                child: ShaderMask(
                  shaderCallback: (r) => const LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                    colors: [Colors.white, Colors.transparent],
                    stops: [0.4, 1],
                  ).createShader(r),
                  blendMode: BlendMode.dstIn,
                  child: Opacity(opacity: 0.45, child: NetImage(server.image(item.backdropPath, size: 'w1280'))),
                ),
              ),
            SafeArea(
              child: FutureBuilder<MediaDetails>(
                future: _details,
                builder: (context, snap) {
                  final d = snap.data;
                  return ListView(
                    padding: pagePadding(context).copyWith(top: 16, bottom: 32),
                    children: [
                      Align(
                        alignment: Alignment.centerLeft,
                        child: IconButton(onPressed: () => context.pop(), icon: const Icon(Icons.arrow_back)),
                      ),
                      SizedBox(height: wide ? 24 : 120),
                      Flex(
                        direction: wide ? Axis.horizontal : Axis.vertical,
                        crossAxisAlignment: wide ? CrossAxisAlignment.start : CrossAxisAlignment.center,
                        children: [
                          AppCard(
                            child: SizedBox(
                              width: wide ? 240 : 160,
                              child: AspectRatio(
                                aspectRatio: 2 / 3,
                                child: NetImage(server.image(item.posterPath, size: 'w500')),
                              ),
                            ),
                          ),
                          SizedBox(width: wide ? 40 : 0, height: wide ? 0 : 20),
                          if (wide)
                            Expanded(
                              child: _Info(item: item, details: d, error: snap.error, onRetry: _retry),
                            )
                          else
                            _Info(item: item, details: d, error: snap.error, onRetry: _retry),
                        ],
                      ),
                    ],
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _retry() => setState(() => _details = getIt<TmdbApi>().details(widget.item));
}

class _Info extends StatelessWidget {
  final MediaItem item;
  final MediaDetails? details;
  final Object? error;
  final VoidCallback onRetry;

  const _Info({required this.item, required this.details, required this.error, required this.onRetry});

  /// "Watch": releases from trackers or trailers from YouTube (as in Lampa).
  Future<void> _chooseWatch(BuildContext context) async {
    final choice = await showDialog<String>(
      context: context,
      builder: (ctx) => SimpleDialog(
        title: Text(item.title),
        children: [
          ListTile(
            autofocus: true,
            leading: const Icon(Icons.download_for_offline_outlined),
            title: Text(ctx.l10n.watchTorrents),
            subtitle: Text(ctx.l10n.watchTorrentsHint),
            onTap: () => Navigator.pop(ctx, 'torrents'),
          ),
          ListTile(
            leading: const Icon(Icons.smart_display_outlined),
            title: Text(ctx.l10n.watchTrailers),
            subtitle: Text(ctx.l10n.watchTrailersHint),
            onTap: () => Navigator.pop(ctx, 'trailers'),
          ),
        ],
      ),
    );
    if (!context.mounted) return;
    switch (choice) {
      case 'torrents':
        context.push('/releases', extra: item);
      case 'trailers':
        context.push('/trailers/${item.isTv ? 'tv' : 'movie'}/${item.id}', extra: item.title);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final d = details;
    final last = getIt<HistoryStore>().lastFor(item);
    final meta = [
      if (item.year != null) '${item.year}',
      if (d != null && d.countries.isNotEmpty) d.countries.take(2).join(', '),
      if (d?.runtime != null) l.detailsMinutes(d!.runtime!),
      if (d?.seasons != null) l.detailsSeasons(d!.seasons!),
    ].join(' · ');

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(item.title, style: AppText.pageTitle),
        if (item.originalTitle.isNotEmpty && item.originalTitle != item.title) ...[
          const SizedBox(height: 4),
          Text(item.originalTitle, style: AppText.body.copyWith(color: AppColors.muted)),
        ],
        const SizedBox(height: 12),
        Wrap(
          spacing: 12,
          runSpacing: 8,
          crossAxisAlignment: WrapCrossAlignment.center,
          children: [
            if (item.rating > 0)
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.star, color: AppColors.warning, size: 16),
                  const SizedBox(width: 6),
                  Text(item.rating.toStringAsFixed(1), style: AppText.statValue),
                ],
              ),
            if (item.isTv) AppBadge(l.badgeSeries, tone: Tone.accent),
            if (meta.isNotEmpty) Text(meta, style: AppText.small),
          ],
        ),
        if (d != null && d.genres.isNotEmpty) ...[
          const SizedBox(height: 8),
          Text(d.genres.join(', '), style: AppText.small),
        ],
        const SizedBox(height: 20),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            FilledButton.icon(
              autofocus: true,
              onPressed: () => _chooseWatch(context),
              icon: const Icon(Icons.play_arrow),
              label: Text(l.detailsWatch),
            ),
            if (last != null)
              OutlinedButton.icon(
                onPressed: () => context.push(
                  '/files',
                  extra: FilesArgs(media: item, hash: last.hash, title: last.torrentTitle, resumeFileId: last.fileId),
                ),
                icon: const Icon(Icons.history),
                label: Text(l.detailsContinue(last.fileName), overflow: TextOverflow.ellipsis),
              ),
          ],
        ),
        SectionTitle(l.detailsOverview, inset: false),
        if (d?.tagline != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Text(l.quoted(d!.tagline!), style: AppText.body.copyWith(color: AppColors.muted)),
          ),
        Text((d?.item.overview.isNotEmpty ?? false) ? d!.item.overview : item.overview, style: AppText.body),
        if (d != null && d.cast.isNotEmpty) ...[
          SectionTitle(l.detailsCast, inset: false),
          Text(d.cast.join(', '), style: AppText.small.copyWith(height: 1.45)),
        ],
        if (error != null) ...[
          const SizedBox(height: 20),
          AppBanner(tone: Tone.error, icon: Icons.error_outline, message: errorText(error!)),
          const SizedBox(height: 12),
          OutlinedButton(onPressed: onRetry, child: Text(l.retry)),
        ],
      ],
    );
  }
}
