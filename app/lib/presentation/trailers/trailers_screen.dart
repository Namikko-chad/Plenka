import 'package:flutter/material.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../../core/utils/format.dart';
import '../../data/api/tmdb_api.dart';
import '../../data/models/media.dart';
import '../../data/models/trailer.dart';
import '../common/widgets/common.dart';
import '../player/playback.dart';
import '../../l10n/l10n.dart';

/// Trailers and other YouTube videos of a movie/series (as in Lampa); they open in YouTube.
class TrailersScreen extends StatefulWidget {
  final MediaType type;
  final int id;
  final String? title;

  const TrailersScreen({super.key, required this.type, required this.id, this.title});

  @override
  State<TrailersScreen> createState() => _TrailersScreenState();
}

class _TrailersScreenState extends State<TrailersScreen> {
  late Future<List<Trailer>> _videos = _load();

  Future<List<Trailer>> _load() => getIt<TmdbApi>().videos(widget.type, widget.id);

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final wide = isWide(context);
    return Scaffold(
      appBar: AppBar(title: Text(widget.title ?? l.trailersTitle)),
      body: PageBackground(
        child: FutureBuilder<List<Trailer>>(
          future: _videos,
          builder: (context, snap) {
            if (snap.hasError) {
              return MessageView.error(snap.error!, onRetry: () => setState(() => _videos = _load()));
            }
            final list = snap.data;
            if (list == null) return LoadingIndicator(message: l.trailersLoading);
            if (list.isEmpty) return MessageView(icon: Icons.videocam_off_outlined, message: l.trailersEmpty);
            return GridView.builder(
              padding: pagePadding(context).copyWith(top: 20, bottom: 32),
              gridDelegate: SliverGridDelegateWithMaxCrossAxisExtent(
                maxCrossAxisExtent: wide ? 360 : 480,
                mainAxisSpacing: 20,
                crossAxisSpacing: 20,
                childAspectRatio: 16 / 9 / 1.45,
              ),
              itemCount: list.length,
              itemBuilder: (_, i) => _TrailerCard(trailer: list[i], autofocus: i == 0),
            );
          },
        ),
      ),
    );
  }
}

class _TrailerCard extends StatelessWidget {
  final Trailer trailer;
  final bool autofocus;
  const _TrailerCard({required this.trailer, required this.autofocus});

  @override
  Widget build(BuildContext context) {
    final t = trailer;
    final meta = [
      t.type,
      if (t.language.isNotEmpty) t.language.toUpperCase(),
      if (t.publishedAt != null) formatDate(t.publishedAt),
    ].join(' · ');
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        FocusCard(
          autofocus: autofocus,
          focusScale: 1.04,
          onTap: () => openYoutube(context, t.watchUrl),
          child: AspectRatio(
            aspectRatio: 16 / 9,
            child: Stack(
              fit: StackFit.expand,
              children: [
                NetImage(t.thumbnail),
                const Center(child: Icon(Icons.play_circle_outline, size: 48, color: AppColors.text)),
              ],
            ),
          ),
        ),
        const SizedBox(height: 8),
        Text(
          t.name,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: AppText.body.copyWith(fontWeight: FontWeight.w600),
        ),
        Text(meta, maxLines: 1, overflow: TextOverflow.ellipsis, style: AppText.small),
      ],
    );
  }
}
