import 'package:equatable/equatable.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../../core/utils/format.dart';
import '../../data/api/search_api.dart';
import '../../data/models/media.dart';
import '../../data/models/torrent.dart';
import '../common/widgets/common.dart';
import '../router/app_router.dart';
import '../../l10n/l10n.dart';

enum ReleaseSort {
  seeders,
  size,
  date;

  String label(AppLocalizations l) => switch (this) {
    seeders => l.releasesSortSeeders,
    size => l.releasesSortSize,
    date => l.releasesSortDate,
  };
}

class ReleasesState extends Equatable {
  final bool loading;
  final List<TorrentResult> all;

  /// How the trackers responded to the last search.
  final List<IndexerStatus> indexers;
  final Quality? quality;
  final ReleaseSort sort;
  final Object? error;

  const ReleasesState({
    this.loading = false,
    this.all = const [],
    this.indexers = const [],
    this.quality,
    this.sort = ReleaseSort.seeders,
    this.error,
  });

  List<TorrentResult> get visible {
    final list = quality == null ? [...all] : all.where((t) => t.quality == quality).toList();
    switch (sort) {
      case ReleaseSort.seeders:
        list.sort((a, b) => b.seeders.compareTo(a.seeders));
      case ReleaseSort.size:
        list.sort((a, b) => b.size.compareTo(a.size));
      case ReleaseSort.date:
        list.sort((a, b) => (b.publishDate ?? DateTime(0)).compareTo(a.publishDate ?? DateTime(0)));
    }
    return list;
  }

  /// Qualities actually present in the results — only those are shown as filters.
  List<Quality> get qualities => Quality.values.where((q) => all.any((t) => t.quality == q)).toList();

  ReleasesState copyWith({
    bool? loading,
    List<TorrentResult>? all,
    List<IndexerStatus>? indexers,
    Quality? Function()? quality,
    ReleaseSort? sort,
    Object? Function()? error,
  }) => ReleasesState(
    loading: loading ?? this.loading,
    all: all ?? this.all,
    indexers: indexers ?? this.indexers,
    quality: quality != null ? quality() : this.quality,
    sort: sort ?? this.sort,
    error: error != null ? error() : this.error,
  );

  @override
  List<Object?> get props => [loading, all, indexers, quality, sort, error];
}

class ReleasesCubit extends Cubit<ReleasesState> {
  final SearchApi _search;
  ReleasesCubit(this._search) : super(const ReleasesState());

  /// Default queries: the Russian and the original title; with the year for movies,
  /// without it for series (season releases come out in different years).
  static List<String> defaultQueries(MediaItem m) {
    final year = (!m.isTv && m.year != null) ? ' ${m.year}' : '';
    return {'${m.title}$year', if (m.originalTitle.isNotEmpty) '${m.originalTitle}$year'}.toList();
  }

  Future<void> search(List<String> queries) async {
    emit(state.copyWith(loading: true, error: () => null));
    try {
      final r = await _search.search(queries);
      emit(state.copyWith(loading: false, all: r.results, indexers: r.indexers, quality: () => null));
    } catch (e) {
      emit(state.copyWith(loading: false, all: const [], indexers: const [], error: () => e));
    }
  }

  void setQuality(Quality? q) => emit(state.copyWith(quality: () => q));
  void setSort(ReleaseSort s) => emit(state.copyWith(sort: s));
}

class ReleasesScreen extends StatefulWidget {
  final MediaItem media;
  const ReleasesScreen({super.key, required this.media});

  @override
  State<ReleasesScreen> createState() => _ReleasesScreenState();
}

class _ReleasesScreenState extends State<ReleasesScreen> {
  late final _cubit = ReleasesCubit(getIt())..search(ReleasesCubit.defaultQueries(widget.media));
  late final _query = TextEditingController(text: ReleasesCubit.defaultQueries(widget.media).first);

  @override
  void dispose() {
    _cubit.close();
    _query.dispose();
    super.dispose();
  }

  /// Tapping a release works as before: a single video starts right away.
  /// [browse] — the "Files" button: just the file list (with preload there), no autostart.
  void _open(TorrentResult t, {bool browse = false}) {
    context.push(
      '/files',
      extra: FilesArgs(media: widget.media, torrent: t, title: t.title, browse: browse),
    );
  }

  @override
  Widget build(BuildContext context) {
    final pad = pagePadding(context);
    return BlocProvider.value(
      value: _cubit,
      child: Scaffold(
        appBar: AppBar(title: Text(widget.media.title)),
        body: PageBackground(
          child: BlocBuilder<ReleasesCubit, ReleasesState>(
            builder: (context, s) {
              final visible = s.visible;
              return Column(
                children: [
                  Padding(
                    padding: pad.copyWith(top: 20),
                    child: TextField(
                      controller: _query,
                      textInputAction: TextInputAction.search,
                      onSubmitted: (q) => _cubit.search([q]),
                      decoration: InputDecoration(
                        prefixIcon: const Icon(Icons.manage_search),
                        hintText: context.l10n.releasesQueryHint,
                      ),
                    ),
                  ),
                  if (!s.loading && s.error == null && s.indexers.isNotEmpty)
                    Padding(
                      padding: pad.copyWith(top: 8),
                      child: _IndexersSummary(indexers: s.indexers),
                    ),
                  const SizedBox(height: 4),
                  SizedBox(
                    height: 48,
                    child: ListView(
                      scrollDirection: Axis.horizontal,
                      padding: pad,
                      children: [
                        Center(
                          child: Segmented<ReleaseSort>(
                            semanticLabel: context.l10n.releasesSort,
                            options: [for (final v in ReleaseSort.values) (v, v.label(context.l10n))],
                            value: s.sort,
                            onChanged: _cubit.setSort,
                          ),
                        ),
                        if (s.qualities.length > 1) ...[
                          const SizedBox(width: 12),
                          Center(
                            child: Segmented<Quality?>(
                              semanticLabel: context.l10n.releasesQuality,
                              options: [
                                (null, context.l10n.releasesQualityAny),
                                for (final q in s.qualities) (q, q.label),
                              ],
                              value: s.quality,
                              onChanged: _cubit.setQuality,
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                  Expanded(
                    child: Builder(
                      builder: (context) {
                        if (s.loading) return LoadingIndicator(message: context.l10n.releasesSearching);
                        if (s.error != null) {
                          return MessageView.error(s.error!, onRetry: () => _cubit.search([_query.text]));
                        }
                        if (visible.isEmpty) {
                          return MessageView(
                            icon: Icons.search_off,
                            message: s.indexers.isEmpty
                                ? context.l10n.releasesNoIndexers
                                : context.l10n.releasesNotFound,
                            actionLabel: context.l10n.releasesSetUpIndexers,
                            onAction: () async {
                              await context.push('/server/indexers');
                              _cubit.search([_query.text]);
                            },
                          );
                        }
                        return ListView.builder(
                          padding: pad.copyWith(top: 8, bottom: 24),
                          itemCount: visible.length,
                          itemBuilder: (_, i) => _ReleaseTile(
                            torrent: visible[i],
                            autofocus: i == 0,
                            onTap: () => _open(visible[i]),
                            onFiles: () => _open(visible[i], browse: true),
                          ),
                        );
                      },
                    ),
                  ),
                ],
              );
            },
          ),
        ),
      ),
    );
  }
}

/// "Trackers: 5 of 6 responded" + the names of those that didn't.
class _IndexersSummary extends StatelessWidget {
  final List<IndexerStatus> indexers;
  const _IndexersSummary({required this.indexers});

  @override
  Widget build(BuildContext context) {
    final failed = indexers.where((i) => !i.ok).toList();
    final ok = indexers.length - failed.length;
    return Row(
      children: [
        SizedBox(
          width: 14,
          child: Center(child: StatusDot(ok: failed.isEmpty)),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            [
              context.l10n.releasesIndexersAnswered(ok, indexers.length),
              if (failed.isNotEmpty)
                context.l10n.releasesIndexersFailed(failed.map((i) => i.name.isEmpty ? i.id : i.name).join(', ')),
            ].join(' · '),
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: AppText.small.copyWith(color: failed.isEmpty ? AppColors.muted : AppColors.warning),
          ),
        ),
      ],
    );
  }
}

class _ReleaseTile extends StatelessWidget {
  final TorrentResult torrent;
  final bool autofocus;
  final VoidCallback onTap;
  final VoidCallback onFiles;

  const _ReleaseTile({required this.torrent, required this.autofocus, required this.onTap, required this.onFiles});

  @override
  Widget build(BuildContext context) {
    final t = torrent;
    final seedColor = t.seeders >= 10 ? AppColors.ok : (t.seeders > 0 ? AppColors.warning : AppColors.error);
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        children: [
          Expanded(child: _card(t, seedColor)),
          const SizedBox(width: 8),
          IconButton(
            tooltip: context.l10n.filesTitle,
            onPressed: onFiles,
            icon: const Icon(Icons.folder_open_outlined),
          ),
        ],
      ),
    );
  }

  Widget _card(TorrentResult t, Color seedColor) {
    return FocusCard(
      autofocus: autofocus,
      focusScale: 1.01,
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    t.title,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: AppText.body.copyWith(height: 1.35),
                  ),
                  const SizedBox(height: 8),
                  Wrap(
                    spacing: 10,
                    runSpacing: 6,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: [
                      AppBadge(t.quality.label, tone: t.quality.index <= 1 ? Tone.accent : Tone.neutral),
                      if (t.isHdr) const AppBadge('HDR', tone: Tone.warning),
                      Text(t.tracker, style: AppText.small),
                      Text(formatBytes(t.size), style: AppText.small),
                      if (t.publishDate != null) Text(formatDate(t.publishDate), style: AppText.small),
                    ],
                  ),
                ],
              ),
            ),
            const SizedBox(width: 12),
            Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(Icons.arrow_upward, size: 14, color: seedColor),
                    const SizedBox(width: 2),
                    Text(
                      '${t.seeders}',
                      style: AppText.body.copyWith(color: seedColor, fontWeight: FontWeight.w700),
                    ),
                  ],
                ),
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(Icons.arrow_downward, size: 14, color: AppColors.muted),
                    const SizedBox(width: 2),
                    Text('${t.peers}', style: AppText.small),
                  ],
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
