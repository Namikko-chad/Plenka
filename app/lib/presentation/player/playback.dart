import 'dart:io';

import 'package:android_intent_plus/android_intent.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/di/injection.dart';
import '../../data/api/torrent_api.dart';
import '../../data/storage/history_store.dart';
import '../../data/storage/settings_store.dart';
import '../common/widgets/common.dart';
import '../router/app_router.dart';
import '../../l10n/l10n.dart';

/// Plays a file in the built-in or an external player, per the setting.
/// [replace] — replace the current screen (when the file list needn't be shown).
Future<void> startPlayback(BuildContext context, PlayerArgs args, {bool replace = false}) async {
  var mode = getIt<SettingsStore>().playerMode;
  if (!Platform.isAndroid) mode = PlayerMode.internal;

  if (mode == PlayerMode.ask) {
    final chosen = await showDialog<PlayerMode>(
      context: context,
      builder: (ctx) => SimpleDialog(
        title: Text(ctx.l10n.playOpenWith),
        children: [
          SimpleDialogOption(
            onPressed: () => Navigator.pop(ctx, PlayerMode.internal),
            child: ListTile(leading: const Icon(Icons.play_circle_outline), title: Text(ctx.l10n.playBuiltInPlayer)),
          ),
          SimpleDialogOption(
            onPressed: () => Navigator.pop(ctx, PlayerMode.external),
            child: ListTile(leading: const Icon(Icons.open_in_new), title: Text(ctx.l10n.playExternalPlayer)),
          ),
        ],
      ),
    );
    if (chosen == null || !context.mounted) return;
    mode = chosen;
  }

  if (mode == PlayerMode.external) {
    await _openExternal(args);
    return;
  }
  if (replace) {
    context.pushReplacement('/player', extra: args);
  } else {
    await context.push('/player', extra: args);
  }
}

Future<void> _openExternal(PlayerArgs args) async {
  final url = getIt<TorrentApi>().streamUrl(args.hash, args.file);
  // The external player's position is unknown, but the entry is needed for "Continue watching".
  final history = getIt<HistoryStore>();
  final prev = history.entry(args.hash, args.file.id);
  await history.save(
    WatchEntry(
      media: args.media,
      hash: args.hash,
      fileId: args.file.id,
      fileName: args.file.name,
      torrentTitle: args.torrentTitle,
      positionMs: prev?.positionMs ?? 0,
      durationMs: prev?.durationMs ?? 0,
      updatedAt: DateTime.now(),
    ),
  );
  await AndroidIntent(
    action: 'action_view',
    data: url,
    type: 'video/*',
    arguments: {'title': args.media?.title ?? args.file.name},
  ).launch();
}

/// Opens a YouTube video in an external app: on Android — a VIEW intent (handled by
/// YouTube, including on Android TV), on Linux — `xdg-open`. On error — a snackbar.
Future<void> openYoutube(BuildContext context, String url) async {
  final messenger = ScaffoldMessenger.of(context);
  try {
    if (Platform.isAndroid) {
      await AndroidIntent(action: 'action_view', data: url).launch();
    } else {
      final r = await Process.run('xdg-open', [url]);
      if (r.exitCode != 0) throw Exception('${r.stderr}'.trim().isEmpty ? 'xdg-open: ${r.exitCode}' : '${r.stderr}');
    }
  } catch (e) {
    messenger.showSnackBar(SnackBar(content: Text(tr.trailersOpenFailed(errorText(e)))));
  }
}
