import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../core/di/injection.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/server/server.dart';
import '../../../data/models/media.dart';
import 'common.dart';
import '../../../l10n/l10n.dart';

/// Poster with title, year and rating.
class PosterCard extends StatelessWidget {
  final MediaItem item;
  final double width;
  final VoidCallback? onTap;
  final bool autofocus;

  const PosterCard({super.key, required this.item, required this.width, this.onTap, this.autofocus = false});

  @override
  Widget build(BuildContext context) {
    final server = getIt<ServerClient>();
    return SizedBox(
      width: width,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          FocusCard(
            autofocus: autofocus,
            onTap: onTap ?? () => context.push('/details', extra: item),
            child: AspectRatio(
              aspectRatio: 2 / 3,
              child: Stack(
                fit: StackFit.expand,
                children: [
                  NetImage(server.image(item.posterPath)),
                  if (item.rating > 0)
                    Positioned(
                      top: 6,
                      right: 6,
                      child: AppBadge(item.rating.toStringAsFixed(1), tone: _ratingTone(item.rating), onImage: true),
                    ),
                  if (item.isTv)
                    Positioned(
                      top: 6,
                      left: 6,
                      child: AppBadge(context.l10n.badgeSeries, tone: Tone.accent, onImage: true),
                    ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 8),
          Text(item.title, maxLines: 2, overflow: TextOverflow.ellipsis, style: AppText.body.copyWith(height: 1.3)),
          if (item.year != null) Text('${item.year}', style: AppText.small),
        ],
      ),
    );
  }
}

Tone _ratingTone(double r) => r >= 7 ? Tone.ok : (r >= 5 ? Tone.warning : Tone.neutral);

/// Horizontal row of posters.
class MediaRow extends StatelessWidget {
  final String title;
  final List<MediaItem> items;
  final bool autofocusFirst;

  const MediaRow({super.key, required this.title, required this.items, this.autofocusFirst = false});

  @override
  Widget build(BuildContext context) {
    final w = posterWidth(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SectionTitle(title),
        SizedBox(
          height: w * 1.5 + 64,
          child: FocusTraversalGroup(
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              padding: pagePadding(context).copyWith(top: 6),
              clipBehavior: Clip.none,
              itemCount: items.length,
              separatorBuilder: (_, _) => const SizedBox(width: 16),
              itemBuilder: (_, i) => PosterCard(item: items[i], width: w, autofocus: autofocusFirst && i == 0),
            ),
          ),
        ),
      ],
    );
  }
}

/// Poster grid (search).
class MediaGrid extends StatelessWidget {
  final List<MediaItem> items;
  const MediaGrid({super.key, required this.items});

  @override
  Widget build(BuildContext context) {
    final w = posterWidth(context);
    return GridView.builder(
      padding: pagePadding(context).copyWith(top: 12, bottom: 24),
      clipBehavior: Clip.none,
      gridDelegate: SliverGridDelegateWithMaxCrossAxisExtent(
        maxCrossAxisExtent: w + 16,
        childAspectRatio: w / (w * 1.5 + 64),
        mainAxisSpacing: 16,
        crossAxisSpacing: 16,
      ),
      itemCount: items.length,
      itemBuilder: (_, i) => PosterCard(item: items[i], width: w),
    );
  }
}
