import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../../core/utils/format.dart';
import '../../data/models/torrent.dart';
import '../../l10n/l10n.dart';
import '../common/widgets/common.dart';
import '../router/app_router.dart';
import 'torrents_cubit.dart';

/// Torrents tab: every torrent saved in the server's engine, like the web Torrents tab.
class TorrentsScreen extends StatelessWidget {
  const TorrentsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocProvider(create: (_) => TorrentsCubit(getIt())..start(), child: const _TorrentsView());
  }
}

class _TorrentsView extends StatelessWidget {
  const _TorrentsView();

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final cubit = context.read<TorrentsCubit>();
    final pad = pagePadding(context);
    return BlocBuilder<TorrentsCubit, TorrentsState>(
      builder: (context, s) {
        final header = Padding(
          padding: pad.copyWith(top: 32, bottom: 16),
          child: Row(
            children: [
              Expanded(child: PageTitle(l.torrentsTitle)),
              OutlinedButton.icon(
                autofocus: s.list.isEmpty,
                onPressed: () => showAddTorrent(context, cubit),
                icon: const Icon(Icons.add, size: 16),
                label: Text(l.torrentsAdd),
              ),
            ],
          ),
        );
        Widget body;
        if (!s.loaded && s.error == null) {
          body = const LoadingIndicator();
        } else if (!s.loaded) {
          body = MessageView.error(s.error!, onRetry: cubit.refresh);
        } else if (s.list.isEmpty) {
          body = MessageView(icon: Icons.download_outlined, message: l.torrentsEmpty);
        } else {
          body = RefreshIndicator(
            onRefresh: cubit.refresh,
            child: ListView.builder(
              padding: pad.copyWith(bottom: 32),
              itemCount: s.list.length + (s.error != null ? 1 : 0),
              itemBuilder: (context, i) {
                if (s.error != null) {
                  if (i == 0) {
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: AppBanner(tone: Tone.error, icon: Icons.error_outline, message: errorText(s.error!)),
                    );
                  }
                  i--;
                }
                final t = s.list[i];
                return _TorrentTile(
                  key: ValueKey(t.hash),
                  torrent: t,
                  autofocus: i == 0,
                  onTap: () => context.push('/torrent', extra: t),
                  onFiles: () => openTorrentFiles(context, t),
                  onDrop: () => dropTorrent(context, cubit.drop, t),
                  onRemove: () => removeTorrent(context, cubit.remove, t),
                );
              },
            ),
          );
        }
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            header,
            Expanded(child: body),
          ],
        );
      },
    );
  }
}

// ─── Shared actions (list and details) ─────────────────────────────────────

/// Badge tone of the engine status.
Tone torrentTone(Torrent t) => switch (t.stat) {
  3 => Tone.ok,
  2 => Tone.warning,
  0 || 1 => Tone.accent,
  _ => Tone.neutral,
};

/// The Torrents tab only shows the files; nothing starts playing by itself.
void openTorrentFiles(BuildContext context, Torrent t) {
  context.push(
    '/files',
    extra: FilesArgs(hash: t.hash, media: t.media, title: t.displayTitle, browse: true),
  );
}

Future<void> dropTorrent(BuildContext context, Future<void> Function(String) drop, Torrent t) async {
  final messenger = ScaffoldMessenger.of(context);
  final l = context.l10n;
  try {
    await drop(t.hash);
    messenger.showSnackBar(SnackBar(content: Text(l.torrentsDropped)));
  } catch (e) {
    messenger.showSnackBar(SnackBar(content: Text(errorText(e))));
  }
}

/// Asks first; true — the torrent was deleted.
Future<bool> removeTorrent(BuildContext context, Future<void> Function(String) remove, Torrent t) async {
  final messenger = ScaffoldMessenger.of(context);
  final l = context.l10n;
  final ok = await showDialog<bool>(
    context: context,
    builder: (ctx) => AlertDialog(
      title: Text(ctx.l10n.torrentsDelete),
      content: Text(ctx.l10n.torrentsDeleteConfirm(t.displayTitle)),
      actions: [
        TextButton(autofocus: true, onPressed: () => Navigator.pop(ctx, false), child: Text(ctx.l10n.cancel)),
        TextButton(
          style: TextButton.styleFrom(foregroundColor: AppColors.error),
          onPressed: () => Navigator.pop(ctx, true),
          child: Text(ctx.l10n.torrentsDelete),
        ),
      ],
    ),
  );
  if (ok != true) return false;
  try {
    await remove(t.hash);
    messenger.showSnackBar(SnackBar(content: Text(l.torrentsDeleted)));
    return true;
  } catch (e) {
    messenger.showSnackBar(SnackBar(content: Text(errorText(e))));
    return false;
  }
}

/// "Add a torrent": a magnet, a .torrent URL or a hash, plus an optional title.
Future<void> showAddTorrent(BuildContext context, TorrentsCubit cubit) async {
  final messenger = ScaffoldMessenger.of(context);
  final l = context.l10n;
  final added = await showDialog<bool>(
    context: context,
    builder: (_) => _AddDialog(cubit: cubit),
  );
  if (added == true) messenger.showSnackBar(SnackBar(content: Text(l.torrentsAdded)));
}

class _AddDialog extends StatefulWidget {
  final TorrentsCubit cubit;
  const _AddDialog({required this.cubit});

  @override
  State<_AddDialog> createState() => _AddDialogState();
}

class _AddDialogState extends State<_AddDialog> {
  final _link = TextEditingController();
  final _title = TextEditingController();
  bool _busy = false;
  Object? _error;

  @override
  void dispose() {
    _link.dispose();
    _title.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final link = _link.text.trim();
    if (_busy || link.isEmpty) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.cubit.add(link: link, title: _title.text.trim());
      if (mounted) Navigator.pop(context, true);
    } catch (e) {
      if (mounted) {
        setState(() {
          _busy = false;
          _error = e;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    return AlertDialog(
      title: Text(l.torrentsAddTitle),
      content: SizedBox(
        width: 480,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            FieldLabel(l.torrentsLink),
            TextField(
              controller: _link,
              autofocus: true,
              textInputAction: TextInputAction.next,
              decoration: const InputDecoration(hintText: 'magnet:?xt=urn:btih:…'),
              onChanged: (_) => setState(() {}),
            ),
            const SizedBox(height: 16),
            FieldLabel(l.torrentsTitleField),
            TextField(controller: _title, textInputAction: TextInputAction.done, onSubmitted: (_) => _submit()),
            if (_error != null) ...[
              const SizedBox(height: 16),
              AppBanner(tone: Tone.error, icon: Icons.error_outline, message: errorText(_error!)),
            ],
          ],
        ),
      ),
      actions: [
        TextButton(onPressed: _busy ? null : () => Navigator.pop(context, false), child: Text(l.cancel)),
        FilledButton(
          onPressed: _busy || _link.text.trim().isEmpty ? null : _submit,
          child: _busy ? const Spinner() : Text(l.torrentsAdd),
        ),
      ],
    );
  }
}

// ─── List tile ─────────────────────────────────────────────────────────────

class _TorrentTile extends StatelessWidget {
  final Torrent torrent;
  final bool autofocus;
  final VoidCallback onTap;
  final VoidCallback onFiles;
  final VoidCallback onDrop;
  final VoidCallback onRemove;

  const _TorrentTile({
    super.key,
    required this.torrent,
    required this.autofocus,
    required this.onTap,
    required this.onFiles,
    required this.onDrop,
    required this.onRemove,
  });

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final t = torrent;
    final full = t.fullProgress;
    final preload = t.preloadProgress;
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        children: [
          Expanded(
            child: FocusCard(
              autofocus: autofocus,
              focusScale: 1.01,
              onTap: onTap,
              child: Padding(
                padding: const EdgeInsets.all(10),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    ClipRRect(
                      borderRadius: BorderRadius.circular(AppRadius.segment),
                      child: SizedBox(width: 54, height: 81, child: NetImage(t.poster.isEmpty ? null : t.poster)),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            t.displayTitle,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: AppText.body.copyWith(height: 1.35),
                          ),
                          const SizedBox(height: 6),
                          Wrap(
                            spacing: 10,
                            runSpacing: 6,
                            crossAxisAlignment: WrapCrossAlignment.center,
                            children: [
                              AppBadge(t.statString.isEmpty ? '—' : t.statString, tone: torrentTone(t)),
                              if (t.torrentSize > 0) Text(formatBytes(t.torrentSize), style: AppText.small),
                              if (full != null)
                                AppBadge(
                                  full >= 1 ? l.torrentsFullDone : '${l.torrentsFull} ${(full * 100).floor()}%',
                                  tone: full >= 1 ? Tone.ok : Tone.accent,
                                ),
                            ],
                          ),
                          if (t.isActive) ...[
                            const SizedBox(height: 6),
                            Text(
                              [
                                '↓ ${formatSpeed(t.downloadSpeed)}',
                                '${l.torrentsPeers.toLowerCase()} ${t.activePeers}/${t.totalPeers}',
                                if (t.connectedSeeders > 0) '${l.torrentsSeeders.toLowerCase()} ${t.connectedSeeders}',
                              ].join('  ·  '),
                              style: AppText.small,
                            ),
                          ],
                          if (preload != null) ...[
                            const SizedBox(height: 8),
                            ProgressLine(value: preload, color: AppColors.warning),
                          ] else if (full != null && full < 1) ...[
                            const SizedBox(height: 8),
                            ProgressLine(value: full),
                          ],
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
          const SizedBox(width: 4),
          Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              IconButton(tooltip: l.filesTitle, onPressed: onFiles, icon: const Icon(Icons.folder_open_outlined)),
              if (t.isActive)
                IconButton(tooltip: l.torrentsDrop, onPressed: onDrop, icon: const Icon(Icons.eject_outlined)),
              IconButton(
                tooltip: l.torrentsDelete,
                onPressed: onRemove,
                icon: const Icon(Icons.delete_outline, color: AppColors.error),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
