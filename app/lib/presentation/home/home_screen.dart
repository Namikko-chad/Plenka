import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../../data/server/server.dart';
import '../../data/storage/history_store.dart';
import '../common/widgets/common.dart';
import '../common/widgets/media_row.dart';
import '../router/app_router.dart';
import 'home_cubit.dart';
import '../../l10n/l10n.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocProvider(
      create: (_) {
        getIt<HistoryStore>().sync(); // pull in what was watched in the web UI
        return HomeCubit(getIt())..load();
      },
      child: BlocBuilder<HomeCubit, HomeState>(
        builder: (context, state) {
          final history = getIt<HistoryStore>();
          return ListenableBuilder(
            listenable: history,
            builder: (context, _) {
              final cont = history.continueWatching();
              if (state.loading && cont.isEmpty) return const LoadingIndicator();
              return RefreshIndicator(
                onRefresh: () => Future.wait([context.read<HomeCubit>().load(), history.sync()]),
                child: ListView(
                  padding: const EdgeInsets.only(bottom: 32),
                  children: [
                    if (cont.isNotEmpty) _ContinueRow(entries: cont),
                    if (state.loading) const Padding(padding: EdgeInsets.all(32), child: LoadingIndicator()),
                    if (state.error != null)
                      SizedBox(
                        height: 360,
                        child: MessageView.error(state.error!, onRetry: context.read<HomeCubit>().load),
                      ),
                    for (final (i, row) in state.rows.indexed)
                      MediaRow(
                        title: row.section.title(context.l10n),
                        items: row.items,
                        autofocusFirst: i == 0 && cont.isEmpty,
                      ),
                  ],
                ),
              );
            },
          );
        },
      ),
    );
  }
}

class _ContinueRow extends StatelessWidget {
  final List<WatchEntry> entries;
  const _ContinueRow({required this.entries});

  @override
  Widget build(BuildContext context) {
    final server = getIt<ServerClient>();
    final w = isWide(context) ? 300.0 : 230.0;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SectionTitle(context.l10n.homeContinue),
        SizedBox(
          height: w * 9 / 16 + 56,
          child: FocusTraversalGroup(
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              padding: pagePadding(context).copyWith(top: 6),
              clipBehavior: Clip.none,
              itemCount: entries.length,
              separatorBuilder: (_, _) => const SizedBox(width: 16),
              itemBuilder: (context, i) {
                final e = entries[i];
                final image = server.image(e.media?.backdropPath, size: 'w780') ?? server.image(e.media?.posterPath);
                return SizedBox(
                  width: w,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      FocusCard(
                        autofocus: i == 0,
                        focusScale: 1.05,
                        onTap: () => context.push(
                          '/files',
                          extra: FilesArgs(media: e.media, hash: e.hash, title: e.torrentTitle, resumeFileId: e.fileId),
                        ),
                        onLongPress: () => _confirmRemove(context, e),
                        child: AspectRatio(
                          aspectRatio: 16 / 9,
                          child: Stack(
                            fit: StackFit.expand,
                            children: [
                              NetImage(image),
                              const Center(child: Icon(Icons.play_circle_outline, size: 48, color: AppColors.text)),
                              Positioned(left: 0, right: 0, bottom: 0, child: ProgressLine(value: e.progress)),
                            ],
                          ),
                        ),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        e.media?.title ?? e.torrentTitle,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: AppText.body.copyWith(fontWeight: FontWeight.w600),
                      ),
                      Text(e.fileName, maxLines: 1, overflow: TextOverflow.ellipsis, style: AppText.small),
                    ],
                  ),
                );
              },
            ),
          ),
        ),
      ],
    );
  }

  Future<void> _confirmRemove(BuildContext context, WatchEntry e) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(ctx.l10n.homeRemoveTitle),
        content: Text(e.media?.title ?? e.torrentTitle),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(ctx.l10n.cancel)),
          OutlinedButton(
            autofocus: true,
            style: AppButtons.of(ButtonVariant.red),
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(ctx.l10n.remove),
          ),
        ],
      ),
    );
    if (ok == true) await getIt<HistoryStore>().remove(e);
  }
}
