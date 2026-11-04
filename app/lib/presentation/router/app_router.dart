import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../data/models/media.dart';
import '../../data/models/torrent.dart';
import '../connect/connect_screen.dart';
import '../connect/connection_cubit.dart';
import '../details/details_screen.dart';
import '../files/files_screen.dart';
import '../home/home_screen.dart';
import '../player/player_screen.dart';
import '../search/search_screen.dart';
import '../server_settings/catalog_screen.dart';
import '../server_settings/engine_screen.dart';
import '../server_settings/indexer_screen.dart';
import '../server_settings/indexers_screen.dart';
import '../settings/settings_screen.dart';
import '../shell/shell.dart';
import '../releases/releases_screen.dart';
import '../torrents/torrent_details_screen.dart';
import '../torrents/torrents_screen.dart';
import '../trailers/trailers_screen.dart';

/// Files screen arguments: either a chosen release (to be added to the torrent engine),
/// or an already known hash (from "Continue watching").
class FilesArgs {
  final MediaItem? media;
  final TorrentResult? torrent;
  final String? hash;
  final String title;

  /// Open the player on this file right away (resuming playback).
  final int? resumeFileId;

  /// Just show the list: all files of the release (unplayable ones muted, not launchable),
  /// without autostarting a single video or resuming playback.
  final bool browse;

  const FilesArgs({this.media, this.torrent, this.hash, required this.title, this.resumeFileId, this.browse = false});
}

class PlayerArgs {
  final String hash;
  final TorrentFile file;

  /// All video files of the release in order — for moving to the next episode.
  final List<TorrentFile> playlist;
  final String torrentTitle;
  final MediaItem? media;

  /// All files of the release — for external subtitles and audio tracks; empty — the player
  /// fetches them.
  final List<TorrentFile> files;

  const PlayerArgs({
    required this.hash,
    required this.file,
    required this.playlist,
    required this.torrentTitle,
    this.media,
    this.files = const [],
  });
}

GoRouter createRouter(ConnectionCubit connection) {
  return GoRouter(
    initialLocation: '/',
    refreshListenable: _StreamListenable(connection.stream),
    redirect: (context, state) {
      final connected = connection.state.isConnected;
      final atConnect = state.matchedLocation == '/connect';
      if (!connected && !atConnect) return '/connect';
      if (connected && atConnect) return '/';
      return null;
    },
    routes: [
      GoRoute(path: '/connect', builder: (_, _) => const ConnectScreen()),
      ShellRoute(
        builder: (_, _, child) => AppShell(child: child),
        routes: [
          GoRoute(
            path: '/',
            pageBuilder: (_, _) => const NoTransitionPage(child: HomeScreen()),
          ),
          GoRoute(
            path: '/search',
            pageBuilder: (_, _) => const NoTransitionPage(child: SearchScreen()),
          ),
          GoRoute(
            path: '/torrents',
            pageBuilder: (_, _) => const NoTransitionPage(child: TorrentsScreen()),
          ),
          GoRoute(
            path: '/settings',
            pageBuilder: (_, _) => const NoTransitionPage(child: SettingsScreen()),
          ),
        ],
      ),
      GoRoute(
        path: '/details',
        builder: (_, s) => DetailsScreen(item: s.extra! as MediaItem),
      ),
      GoRoute(
        path: '/releases',
        builder: (_, s) => ReleasesScreen(media: s.extra! as MediaItem),
      ),
      GoRoute(
        path: '/torrent',
        builder: (_, s) => TorrentDetailsScreen(torrent: s.extra! as Torrent),
      ),
      GoRoute(
        path: '/files',
        builder: (_, s) => FilesScreen(args: s.extra! as FilesArgs),
      ),
      // YouTube videos; extra — the title for the header (optional)
      GoRoute(
        path: '/trailers/:type/:id',
        builder: (_, s) => TrailersScreen(
          type: s.pathParameters['type'] == 'tv' ? MediaType.tv : MediaType.movie,
          id: int.parse(s.pathParameters['id']!),
          title: s.extra as String?,
        ),
      ),
      GoRoute(
        path: '/player',
        builder: (_, s) => PlayerScreen(args: s.extra! as PlayerArgs),
      ),
      // server settings
      GoRoute(path: '/server/catalog', builder: (_, _) => const CatalogSettingsScreen()),
      GoRoute(path: '/server/engine', builder: (_, _) => const EngineSettingsScreen()),
      GoRoute(
        path: '/server/indexers',
        builder: (_, _) => const IndexersScreen(),
        routes: [
          GoRoute(
            path: ':id',
            builder: (_, s) => IndexerScreen(id: s.pathParameters['id']!),
          ),
        ],
      ),
    ],
  );
}

class _StreamListenable extends ChangeNotifier {
  late final StreamSubscription<dynamic> _sub;
  _StreamListenable(Stream<dynamic> stream) {
    _sub = stream.listen((_) => notifyListeners());
  }

  @override
  void dispose() {
    _sub.cancel();
    super.dispose();
  }
}
