// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get tmdbLanguage => 'en-US';

  @override
  String get navHome => 'HOME';

  @override
  String get navSearch => 'SEARCH';

  @override
  String get navSettings => 'SETTINGS';

  @override
  String get cancel => 'Cancel';

  @override
  String get retry => 'Retry';

  @override
  String get remove => 'Remove';

  @override
  String get statusOk => 'OK';

  @override
  String get statusError => 'Error';

  @override
  String quoted(String text) {
    return '“$text”';
  }

  @override
  String get badgeSeries => 'Series';

  @override
  String get routerAddress => 'Router address';

  @override
  String get byteUnits => 'B KB MB GB TB';

  @override
  String perSecond(String size) {
    return '$size/s';
  }

  @override
  String seconds(String value) {
    return '$value s';
  }

  @override
  String get errorTmdbNotConfigured =>
      'TMDB is not configured: set the key in Settings → Catalog and search';

  @override
  String get errorTmdbKeyRejected =>
      'TMDB rejected the key: check it in Settings → Catalog and search';

  @override
  String errorServerMessage(int code, String message) {
    return 'Server error ($code): $message';
  }

  @override
  String errorServerCode(String code) {
    return 'Server error: $code';
  }

  @override
  String get errorServerSlow => 'The server takes too long to respond';

  @override
  String get errorServerUnreachable => 'No connection to the server';

  @override
  String get connectSearching => 'Looking for the server on the router…';

  @override
  String get connectNoResponse => 'The server does not respond at this address';

  @override
  String get connectNotFound => 'Server not found';

  @override
  String get connectNotFoundHint =>
      'Make sure the container on the router is running and this device is on the home network. Or enter the router address.';

  @override
  String get connectSearchAgain => 'Search again';

  @override
  String connectAddressHint(int port) {
    return '192.168.88.1 or https://… (port $port by default)';
  }

  @override
  String get connectButton => 'Connect';

  @override
  String get homeTrending => 'Trending';

  @override
  String get homeNowPlaying => 'Now in theaters';

  @override
  String get homePopularMovies => 'Popular movies';

  @override
  String get homePopularTv => 'Popular TV shows';

  @override
  String get homeTopMovies => 'Top rated movies';

  @override
  String get homeTopTv => 'Top rated TV shows';

  @override
  String get homeContinue => 'Continue watching';

  @override
  String get homeRemoveTitle => 'Remove from “Continue watching”?';

  @override
  String get searchHint => 'Movie or TV show';

  @override
  String get searchEnterTitle => 'Enter a title';

  @override
  String get searchNothingFound => 'Nothing found';

  @override
  String detailsMinutes(int count) {
    return '$count min';
  }

  @override
  String detailsSeasons(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count seasons',
      one: '1 season',
    );
    return '$_temp0';
  }

  @override
  String get detailsWatch => 'Watch';

  @override
  String detailsContinue(String name) {
    return 'Continue: $name';
  }

  @override
  String get detailsOverview => 'Overview';

  @override
  String get detailsCast => 'Cast';

  @override
  String get releasesSortSeeders => 'By seeders';

  @override
  String get releasesSortSize => 'By size';

  @override
  String get releasesSortDate => 'By date';

  @override
  String get releasesQueryHint => 'Custom tracker query';

  @override
  String get releasesSort => 'Sort';

  @override
  String get releasesQuality => 'Quality';

  @override
  String get releasesQualityAny => 'Any';

  @override
  String get releasesSearching => 'Searching trackers for releases…';

  @override
  String get releasesNotFound =>
      'No releases found.\nTry another query or enable more trackers.';

  @override
  String releasesIndexersAnswered(int ok, int total) {
    return 'Trackers: $ok of $total responded';
  }

  @override
  String releasesIndexersFailed(String names) {
    return 'not responding: $names';
  }

  @override
  String get filesTitle => 'Files';

  @override
  String get filesConnectingPeers => 'Connecting to peers…';

  @override
  String get filesFetchingList => 'Fetching the file list from peers…';

  @override
  String get filesAddFailed => 'The server could not add the release';

  @override
  String get filesNoPeers => 'Could not get the file list: no peers';

  @override
  String get playOpenWith => 'Open with?';

  @override
  String get playBuiltInPlayer => 'Built-in player';

  @override
  String get playExternalPlayer => 'External player';

  @override
  String get playerModeInternal => 'Built-in';

  @override
  String get playerModeExternal => 'External (VLC, MX Player…)';

  @override
  String get playerModeAsk => 'Ask every time';

  @override
  String get playerAudioTrack => 'Audio track';

  @override
  String get playerSubtitles => 'Subtitles';

  @override
  String get playerSubtitlesOff => 'Off';

  @override
  String playerTrack(String id) {
    return 'Track $id';
  }

  @override
  String get playerNoTracks => 'No tracks';

  @override
  String playerPeers(String speed, String peers) {
    return '$speed · peers $peers';
  }

  @override
  String get playerPause => 'Pause';

  @override
  String get playerPlay => 'Play';

  @override
  String get playerAudio => 'Audio';

  @override
  String get playerNext => 'Next';

  @override
  String get settingsTitle => 'Settings';

  @override
  String get settingsServer => 'Server';

  @override
  String get settingsServerAddress => 'Server address';

  @override
  String get settingsConnect => 'Connect';

  @override
  String get settingsNotConnected => 'not connected';

  @override
  String get settingsTorrents => 'Torrents';

  @override
  String get settingsTorrentsOk => 'torrent playback';

  @override
  String get settingsTorrentsOff => 'torrent engine is not running';

  @override
  String get settingsIndexers => 'Trackers';

  @override
  String settingsIndexersCount(int enabled, int total) {
    return '$enabled of $total enabled';
  }

  @override
  String get settingsTmdbOk => 'movie catalog';

  @override
  String get settingsTmdbOff => 'no key — set it in Catalog and search';

  @override
  String get settingsPlayer => 'Player';

  @override
  String get settingsLanguage => 'Language';

  @override
  String get languageSystem => 'Device language';

  @override
  String get save => 'Save';

  @override
  String get saved => 'Saved';

  @override
  String get on => 'On';

  @override
  String get off => 'Off';

  @override
  String connectConnectingTo(String address) {
    return 'Connecting to $address…';
  }

  @override
  String connectSavedNotResponding(String address) {
    return 'The server $address does not respond. Make sure it is running and this device is on the same network, or choose another server.';
  }

  @override
  String get connectSearchAuto => 'Search automatically';

  @override
  String get releasesNoIndexers => 'No trackers are enabled.';

  @override
  String get releasesSetUpIndexers => 'Set up trackers';

  @override
  String get settingsChangeServer => 'Change server';

  @override
  String get settingsChangeServerHint =>
      'Enter the address of another plenka-server';

  @override
  String get settingsServerManual => 'set manually';

  @override
  String get settingsServerAuto => 'found automatically';

  @override
  String get settingsFindAuto => 'Find the server automatically';

  @override
  String get settingsServerSettings => 'Server settings';

  @override
  String get settingsCatalog => 'Catalog and search';

  @override
  String get settingsCatalogHint => 'TMDB key, search timeout, FlareSolverr';

  @override
  String get settingsEngine => 'Torrent engine';

  @override
  String get settingsEngineHint =>
      'Cache, connections, speed limits, protocols';

  @override
  String get settingsIndexersHint => 'Enable trackers, logins, definitions';

  @override
  String get catalogTmdbKey => 'TMDB API key';

  @override
  String get catalogTmdbKeyHint => 'v3 API key or v4 token';

  @override
  String get catalogTmdbKeySet =>
      'The key is set. Enter a new one to replace it.';

  @override
  String get catalogTmdbKeyRemove => 'Remove key';

  @override
  String get catalogTmdbKeyRemoved => 'Key removed';

  @override
  String get catalogSearchTimeout => 'Tracker search timeout, s';

  @override
  String get catalogFlareSolverr => 'FlareSolverr address';

  @override
  String get catalogFlareSolverrHint =>
      'For trackers behind a Cloudflare check, e.g. http://192.168.88.1:8191. Empty — off.';

  @override
  String get engineCacheSize => 'Cache size, MB';

  @override
  String get engineReadAhead => 'Read-ahead, %';

  @override
  String get enginePreload => 'Preload to disk, % of every file';

  @override
  String get engineConnections => 'Connection limit';

  @override
  String get engineDownLimit => 'Download limit, KB/s (0 — unlimited)';

  @override
  String get engineUpLimit => 'Upload limit, KB/s (0 — unlimited)';

  @override
  String get engineDisconnect => 'Unload an idle torrent after, s';

  @override
  String get enginePort => 'Incoming port (0 — random)';

  @override
  String get engineRetrackers => 'Retrackers';

  @override
  String get engineRetrackersNone => 'Leave';

  @override
  String get engineRetrackersAdd => 'Add';

  @override
  String get engineRetrackersRemove => 'Remove';

  @override
  String get engineRetrackersReplace => 'Replace';

  @override
  String get engineUseDisk => 'Keep the cache on disk';

  @override
  String get engineSavePath => 'Downloads folder (empty: the data folder)';

  @override
  String get engineRemoveCacheOnDrop =>
      'Delete the cache when a torrent is unloaded';

  @override
  String get engineUpload => 'Upload (seed)';

  @override
  String get engineEncrypt => 'Encrypted connections only';

  @override
  String get engineResponsive => 'Responsive mode (read without piece checks)';

  @override
  String get engineDebug => 'Debug log';

  @override
  String get engineReset => 'Reset to defaults';

  @override
  String get engineResetConfirm =>
      'Reset the torrent engine settings to their defaults?';

  @override
  String get engineResetDone => 'Settings reset';

  @override
  String get engineRestartHint =>
      'The engine restarts after saving; playing torrents reconnect.';

  @override
  String get indexersSearch => 'Find a tracker';

  @override
  String get indexersAll => 'All';

  @override
  String get indexersEnabled => 'Enabled';

  @override
  String get indexersPublic => 'Public';

  @override
  String get indexersPrivate => 'Private';

  @override
  String get indexersTypePublic => 'public';

  @override
  String get indexersTypeSemi => 'semi-private';

  @override
  String get indexersTypePrivate => 'private';

  @override
  String get indexersNeedsSetup => 'needs setup';

  @override
  String get indexersError => 'error';

  @override
  String get indexersEmpty => 'No trackers found';

  @override
  String indexersCounter(int enabled, int total) {
    return '$enabled of $total enabled';
  }

  @override
  String get indexersDefinitions => 'Tracker definitions';

  @override
  String indexersDefinitionsInfo(int count, String date) {
    return 'Loaded: $count · updated $date';
  }

  @override
  String indexersDefinitionsNever(int count) {
    return 'Loaded: $count · not updated yet';
  }

  @override
  String get indexersDefinitionsUpdate => 'Update definitions';

  @override
  String get indexersDefinitionsUpdating => 'Definitions are downloading…';

  @override
  String indexersDefinitionsUpdated(int count) {
    return 'Definitions updated: $count';
  }

  @override
  String indexersDefinitionsFailed(String error) {
    return 'The last update failed: $error';
  }

  @override
  String get indexersDefinitionsEmpty =>
      'The server has no tracker definitions yet. Press “Update definitions” to download them.';

  @override
  String get indexerEnabled => 'Tracker enabled';

  @override
  String get indexerSiteLink => 'Site address';

  @override
  String get indexerCustomLink => 'Custom address';

  @override
  String get indexerSettings => 'Settings';

  @override
  String get indexerNoSettings => 'This tracker has no settings';

  @override
  String get indexerPasswordSet => 'set — leave empty to keep it';

  @override
  String get indexerTest => 'Test';

  @override
  String indexerTestOk(int results, int ms) {
    return 'Works: $results results in $ms ms';
  }

  @override
  String indexerTestFail(String error) {
    return 'Error: $error';
  }

  @override
  String get indexerLastError => 'Last error';

  @override
  String get indexerSaved => 'Tracker settings saved';

  @override
  String get settingsVideoOutput => 'Video output';

  @override
  String get videoOutputHardware => 'Hardware — MediaCodec (HDR, faster)';

  @override
  String get videoOutputHardwareHint =>
      'Decodes straight to the screen: 10-bit, HDR10, HDR10+ and Dolby Vision';

  @override
  String get videoOutputCompatible => 'Compatible — OpenGL';

  @override
  String get videoOutputCompatibleHint =>
      'Try it if the picture is black, glitchy or does not start';

  @override
  String get playerVideoOutputHint =>
      'Playback problem? Try Settings → Video output → Compatible — OpenGL';

  @override
  String get watchTorrents => 'Torrents';

  @override
  String get watchTorrentsHint => 'Find a release on the trackers';

  @override
  String get watchTrailers => 'Trailers';

  @override
  String get watchTrailersHint => 'Trailers and videos on YouTube';

  @override
  String get trailersTitle => 'Trailers';

  @override
  String get trailersLoading => 'Loading videos…';

  @override
  String get trailersEmpty => 'No trailers';

  @override
  String trailersOpenFailed(String error) {
    return 'Could not open YouTube: $error';
  }

  @override
  String get preload => 'Preload';

  @override
  String preloadStarted(String name) {
    return 'Preload started: $name';
  }

  @override
  String preloadProgress(String done, String total) {
    return 'Preload: $done / $total';
  }

  @override
  String get downloadAll => 'Download in full';

  @override
  String get downloadAllCancel => 'Cancel full download';

  @override
  String get downloadAllStarted =>
      'Full download started — the torrent stays on the server\'s disk';

  @override
  String get downloadAllCancelled => 'Full download cancelled';

  @override
  String get downloadAllNoRoom =>
      'The torrent does not fit: the server disk lacks free space';

  @override
  String downloadAllProgress(String done, String total) {
    return 'In full: $done / $total';
  }

  @override
  String downloadAllDone(String total) {
    return 'Downloaded in full: $total';
  }

  @override
  String get navTorrents => 'TORRENTS';

  @override
  String get torrentsTitle => 'Torrents';

  @override
  String get torrentsAdd => 'Add';

  @override
  String get torrentsAddTitle => 'Add a torrent';

  @override
  String get torrentsLink => 'Magnet, .torrent link or hash';

  @override
  String get torrentsTitleField => 'Title (optional)';

  @override
  String get torrentsAdded => 'Torrent added';

  @override
  String get torrentsEmpty =>
      'No torrents. Add a release from search or by a magnet link.';

  @override
  String get torrentsDrop => 'Unload from memory';

  @override
  String get torrentsDropped => 'Unloaded from memory';

  @override
  String get torrentsDelete => 'Delete';

  @override
  String torrentsDeleteConfirm(String title) {
    return 'Delete “$title” from the torrent list?';
  }

  @override
  String get torrentsDeleted => 'Torrent deleted';

  @override
  String get torrentsDetails => 'Details';

  @override
  String get torrentsStatus => 'Status';

  @override
  String get torrentsSize => 'Size';

  @override
  String get torrentsLoaded => 'Loaded';

  @override
  String get torrentsSpeedDown => 'Download';

  @override
  String get torrentsSpeedUp => 'Upload';

  @override
  String get torrentsPeers => 'Peers';

  @override
  String get torrentsSeeders => 'Seeders';

  @override
  String get torrentsFull => 'Full';

  @override
  String get torrentsFullDone => 'Downloaded in full';

  @override
  String get torrentsHash => 'Hash';

  @override
  String get torrentsHashCopied => 'Hash copied';

  @override
  String get torrentsAddedAt => 'Added';

  @override
  String get torrentsCache => 'Cache';

  @override
  String torrentsCacheSummary(
    String filled,
    String capacity,
    int count,
    String pieceSize,
  ) {
    return 'Filled $filled / $capacity · pieces: $count · piece size $pieceSize';
  }

  @override
  String get torrentsNoCache =>
      'The cache is empty — the torrent is not active';

  @override
  String get torrentsLegendDone => 'loaded';

  @override
  String get torrentsLegendPartial => 'partial';

  @override
  String get torrentsLegendQueued => 'queued';

  @override
  String get torrentsLegendReader => 'reading';
}
