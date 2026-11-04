import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_ru.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('ru'),
  ];

  /// TMDB `language` parameter used for catalog requests in this UI language.
  ///
  /// In en, this message translates to:
  /// **'en-US'**
  String get tmdbLanguage;

  /// No description provided for @navHome.
  ///
  /// In en, this message translates to:
  /// **'HOME'**
  String get navHome;

  /// No description provided for @navSearch.
  ///
  /// In en, this message translates to:
  /// **'SEARCH'**
  String get navSearch;

  /// No description provided for @navSettings.
  ///
  /// In en, this message translates to:
  /// **'SETTINGS'**
  String get navSettings;

  /// No description provided for @cancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get cancel;

  /// No description provided for @retry.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get retry;

  /// No description provided for @remove.
  ///
  /// In en, this message translates to:
  /// **'Remove'**
  String get remove;

  /// No description provided for @statusOk.
  ///
  /// In en, this message translates to:
  /// **'OK'**
  String get statusOk;

  /// No description provided for @statusError.
  ///
  /// In en, this message translates to:
  /// **'Error'**
  String get statusError;

  /// No description provided for @quoted.
  ///
  /// In en, this message translates to:
  /// **'“{text}”'**
  String quoted(String text);

  /// No description provided for @badgeSeries.
  ///
  /// In en, this message translates to:
  /// **'Series'**
  String get badgeSeries;

  /// No description provided for @routerAddress.
  ///
  /// In en, this message translates to:
  /// **'Router address'**
  String get routerAddress;

  /// Space-separated size units, smallest first.
  ///
  /// In en, this message translates to:
  /// **'B KB MB GB TB'**
  String get byteUnits;

  /// No description provided for @perSecond.
  ///
  /// In en, this message translates to:
  /// **'{size}/s'**
  String perSecond(String size);

  /// No description provided for @seconds.
  ///
  /// In en, this message translates to:
  /// **'{value} s'**
  String seconds(String value);

  /// No description provided for @errorTmdbNotConfigured.
  ///
  /// In en, this message translates to:
  /// **'TMDB is not configured: set the key in Settings → Catalog and search'**
  String get errorTmdbNotConfigured;

  /// No description provided for @errorTmdbKeyRejected.
  ///
  /// In en, this message translates to:
  /// **'TMDB rejected the key: check it in Settings → Catalog and search'**
  String get errorTmdbKeyRejected;

  /// No description provided for @errorServerMessage.
  ///
  /// In en, this message translates to:
  /// **'Server error ({code}): {message}'**
  String errorServerMessage(int code, String message);

  /// No description provided for @errorServerCode.
  ///
  /// In en, this message translates to:
  /// **'Server error: {code}'**
  String errorServerCode(String code);

  /// No description provided for @errorServerSlow.
  ///
  /// In en, this message translates to:
  /// **'The server takes too long to respond'**
  String get errorServerSlow;

  /// No description provided for @errorServerUnreachable.
  ///
  /// In en, this message translates to:
  /// **'No connection to the server'**
  String get errorServerUnreachable;

  /// No description provided for @connectSearching.
  ///
  /// In en, this message translates to:
  /// **'Looking for the server on the router…'**
  String get connectSearching;

  /// No description provided for @connectNoResponse.
  ///
  /// In en, this message translates to:
  /// **'The server does not respond at this address'**
  String get connectNoResponse;

  /// No description provided for @connectNotFound.
  ///
  /// In en, this message translates to:
  /// **'Server not found'**
  String get connectNotFound;

  /// No description provided for @connectNotFoundHint.
  ///
  /// In en, this message translates to:
  /// **'Make sure the container on the router is running and this device is on the home network. Or enter the router address.'**
  String get connectNotFoundHint;

  /// No description provided for @connectSearchAgain.
  ///
  /// In en, this message translates to:
  /// **'Search again'**
  String get connectSearchAgain;

  /// No description provided for @connectAddressHint.
  ///
  /// In en, this message translates to:
  /// **'192.168.88.1 or https://… (port {port} by default)'**
  String connectAddressHint(int port);

  /// No description provided for @connectButton.
  ///
  /// In en, this message translates to:
  /// **'Connect'**
  String get connectButton;

  /// No description provided for @homeTrending.
  ///
  /// In en, this message translates to:
  /// **'Trending'**
  String get homeTrending;

  /// No description provided for @homeNowPlaying.
  ///
  /// In en, this message translates to:
  /// **'Now in theaters'**
  String get homeNowPlaying;

  /// No description provided for @homePopularMovies.
  ///
  /// In en, this message translates to:
  /// **'Popular movies'**
  String get homePopularMovies;

  /// No description provided for @homePopularTv.
  ///
  /// In en, this message translates to:
  /// **'Popular TV shows'**
  String get homePopularTv;

  /// No description provided for @homeTopMovies.
  ///
  /// In en, this message translates to:
  /// **'Top rated movies'**
  String get homeTopMovies;

  /// No description provided for @homeTopTv.
  ///
  /// In en, this message translates to:
  /// **'Top rated TV shows'**
  String get homeTopTv;

  /// No description provided for @homeContinue.
  ///
  /// In en, this message translates to:
  /// **'Continue watching'**
  String get homeContinue;

  /// No description provided for @homeRemoveTitle.
  ///
  /// In en, this message translates to:
  /// **'Remove from “Continue watching”?'**
  String get homeRemoveTitle;

  /// No description provided for @searchHint.
  ///
  /// In en, this message translates to:
  /// **'Movie or TV show'**
  String get searchHint;

  /// No description provided for @searchEnterTitle.
  ///
  /// In en, this message translates to:
  /// **'Enter a title'**
  String get searchEnterTitle;

  /// No description provided for @searchNothingFound.
  ///
  /// In en, this message translates to:
  /// **'Nothing found'**
  String get searchNothingFound;

  /// No description provided for @detailsMinutes.
  ///
  /// In en, this message translates to:
  /// **'{count} min'**
  String detailsMinutes(int count);

  /// No description provided for @detailsSeasons.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, one{1 season} other{{count} seasons}}'**
  String detailsSeasons(int count);

  /// No description provided for @detailsWatch.
  ///
  /// In en, this message translates to:
  /// **'Watch'**
  String get detailsWatch;

  /// No description provided for @detailsContinue.
  ///
  /// In en, this message translates to:
  /// **'Continue: {name}'**
  String detailsContinue(String name);

  /// No description provided for @detailsOverview.
  ///
  /// In en, this message translates to:
  /// **'Overview'**
  String get detailsOverview;

  /// No description provided for @detailsCast.
  ///
  /// In en, this message translates to:
  /// **'Cast'**
  String get detailsCast;

  /// No description provided for @releasesSortSeeders.
  ///
  /// In en, this message translates to:
  /// **'By seeders'**
  String get releasesSortSeeders;

  /// No description provided for @releasesSortSize.
  ///
  /// In en, this message translates to:
  /// **'By size'**
  String get releasesSortSize;

  /// No description provided for @releasesSortDate.
  ///
  /// In en, this message translates to:
  /// **'By date'**
  String get releasesSortDate;

  /// No description provided for @releasesQueryHint.
  ///
  /// In en, this message translates to:
  /// **'Custom tracker query'**
  String get releasesQueryHint;

  /// No description provided for @releasesSort.
  ///
  /// In en, this message translates to:
  /// **'Sort'**
  String get releasesSort;

  /// No description provided for @releasesQuality.
  ///
  /// In en, this message translates to:
  /// **'Quality'**
  String get releasesQuality;

  /// No description provided for @releasesQualityAny.
  ///
  /// In en, this message translates to:
  /// **'Any'**
  String get releasesQualityAny;

  /// No description provided for @releasesSearching.
  ///
  /// In en, this message translates to:
  /// **'Searching trackers for releases…'**
  String get releasesSearching;

  /// No description provided for @releasesNotFound.
  ///
  /// In en, this message translates to:
  /// **'No releases found.\nTry another query or enable more trackers.'**
  String get releasesNotFound;

  /// No description provided for @releasesIndexersAnswered.
  ///
  /// In en, this message translates to:
  /// **'Trackers: {ok} of {total} responded'**
  String releasesIndexersAnswered(int ok, int total);

  /// No description provided for @releasesIndexersFailed.
  ///
  /// In en, this message translates to:
  /// **'not responding: {names}'**
  String releasesIndexersFailed(String names);

  /// No description provided for @filesTitle.
  ///
  /// In en, this message translates to:
  /// **'Files'**
  String get filesTitle;

  /// No description provided for @filesConnectingPeers.
  ///
  /// In en, this message translates to:
  /// **'Connecting to peers…'**
  String get filesConnectingPeers;

  /// No description provided for @filesFetchingList.
  ///
  /// In en, this message translates to:
  /// **'Fetching the file list from peers…'**
  String get filesFetchingList;

  /// No description provided for @filesAddFailed.
  ///
  /// In en, this message translates to:
  /// **'The server could not add the release'**
  String get filesAddFailed;

  /// No description provided for @filesNoPeers.
  ///
  /// In en, this message translates to:
  /// **'Could not get the file list: no peers'**
  String get filesNoPeers;

  /// No description provided for @playOpenWith.
  ///
  /// In en, this message translates to:
  /// **'Open with?'**
  String get playOpenWith;

  /// No description provided for @playBuiltInPlayer.
  ///
  /// In en, this message translates to:
  /// **'Built-in player'**
  String get playBuiltInPlayer;

  /// No description provided for @playExternalPlayer.
  ///
  /// In en, this message translates to:
  /// **'External player'**
  String get playExternalPlayer;

  /// No description provided for @playerModeInternal.
  ///
  /// In en, this message translates to:
  /// **'Built-in'**
  String get playerModeInternal;

  /// No description provided for @playerModeExternal.
  ///
  /// In en, this message translates to:
  /// **'External (VLC, MX Player…)'**
  String get playerModeExternal;

  /// No description provided for @playerModeAsk.
  ///
  /// In en, this message translates to:
  /// **'Ask every time'**
  String get playerModeAsk;

  /// No description provided for @playerAudioTrack.
  ///
  /// In en, this message translates to:
  /// **'Audio track'**
  String get playerAudioTrack;

  /// No description provided for @playerSubtitles.
  ///
  /// In en, this message translates to:
  /// **'Subtitles'**
  String get playerSubtitles;

  /// No description provided for @playerSubtitlesOff.
  ///
  /// In en, this message translates to:
  /// **'Off'**
  String get playerSubtitlesOff;

  /// No description provided for @playerTrack.
  ///
  /// In en, this message translates to:
  /// **'Track {id}'**
  String playerTrack(String id);

  /// No description provided for @playerNoTracks.
  ///
  /// In en, this message translates to:
  /// **'No tracks'**
  String get playerNoTracks;

  /// No description provided for @playerPeers.
  ///
  /// In en, this message translates to:
  /// **'{speed} · peers {peers}'**
  String playerPeers(String speed, String peers);

  /// No description provided for @playerPause.
  ///
  /// In en, this message translates to:
  /// **'Pause'**
  String get playerPause;

  /// No description provided for @playerPlay.
  ///
  /// In en, this message translates to:
  /// **'Play'**
  String get playerPlay;

  /// No description provided for @playerAudio.
  ///
  /// In en, this message translates to:
  /// **'Audio'**
  String get playerAudio;

  /// No description provided for @playerNext.
  ///
  /// In en, this message translates to:
  /// **'Next'**
  String get playerNext;

  /// No description provided for @settingsTitle.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settingsTitle;

  /// No description provided for @settingsServer.
  ///
  /// In en, this message translates to:
  /// **'Server'**
  String get settingsServer;

  /// No description provided for @settingsServerAddress.
  ///
  /// In en, this message translates to:
  /// **'Server address'**
  String get settingsServerAddress;

  /// No description provided for @settingsConnect.
  ///
  /// In en, this message translates to:
  /// **'Connect'**
  String get settingsConnect;

  /// No description provided for @settingsNotConnected.
  ///
  /// In en, this message translates to:
  /// **'not connected'**
  String get settingsNotConnected;

  /// No description provided for @settingsTorrents.
  ///
  /// In en, this message translates to:
  /// **'Torrents'**
  String get settingsTorrents;

  /// No description provided for @settingsTorrentsOk.
  ///
  /// In en, this message translates to:
  /// **'torrent playback'**
  String get settingsTorrentsOk;

  /// No description provided for @settingsTorrentsOff.
  ///
  /// In en, this message translates to:
  /// **'torrent engine is not running'**
  String get settingsTorrentsOff;

  /// No description provided for @settingsIndexers.
  ///
  /// In en, this message translates to:
  /// **'Trackers'**
  String get settingsIndexers;

  /// No description provided for @settingsIndexersCount.
  ///
  /// In en, this message translates to:
  /// **'{enabled} of {total} enabled'**
  String settingsIndexersCount(int enabled, int total);

  /// No description provided for @settingsTmdbOk.
  ///
  /// In en, this message translates to:
  /// **'movie catalog'**
  String get settingsTmdbOk;

  /// No description provided for @settingsTmdbOff.
  ///
  /// In en, this message translates to:
  /// **'no key — set it in Catalog and search'**
  String get settingsTmdbOff;

  /// No description provided for @settingsPlayer.
  ///
  /// In en, this message translates to:
  /// **'Player'**
  String get settingsPlayer;

  /// No description provided for @settingsLanguage.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get settingsLanguage;

  /// No description provided for @languageSystem.
  ///
  /// In en, this message translates to:
  /// **'Device language'**
  String get languageSystem;

  /// No description provided for @save.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get save;

  /// No description provided for @saved.
  ///
  /// In en, this message translates to:
  /// **'Saved'**
  String get saved;

  /// No description provided for @on.
  ///
  /// In en, this message translates to:
  /// **'On'**
  String get on;

  /// No description provided for @off.
  ///
  /// In en, this message translates to:
  /// **'Off'**
  String get off;

  /// No description provided for @connectConnectingTo.
  ///
  /// In en, this message translates to:
  /// **'Connecting to {address}…'**
  String connectConnectingTo(String address);

  /// No description provided for @connectSavedNotResponding.
  ///
  /// In en, this message translates to:
  /// **'The server {address} does not respond. Make sure it is running and this device is on the same network, or choose another server.'**
  String connectSavedNotResponding(String address);

  /// No description provided for @connectSearchAuto.
  ///
  /// In en, this message translates to:
  /// **'Search automatically'**
  String get connectSearchAuto;

  /// No description provided for @releasesNoIndexers.
  ///
  /// In en, this message translates to:
  /// **'No trackers are enabled.'**
  String get releasesNoIndexers;

  /// No description provided for @releasesSetUpIndexers.
  ///
  /// In en, this message translates to:
  /// **'Set up trackers'**
  String get releasesSetUpIndexers;

  /// No description provided for @settingsChangeServer.
  ///
  /// In en, this message translates to:
  /// **'Change server'**
  String get settingsChangeServer;

  /// No description provided for @settingsChangeServerHint.
  ///
  /// In en, this message translates to:
  /// **'Enter the address of another plenka-server'**
  String get settingsChangeServerHint;

  /// No description provided for @settingsServerManual.
  ///
  /// In en, this message translates to:
  /// **'set manually'**
  String get settingsServerManual;

  /// No description provided for @settingsServerAuto.
  ///
  /// In en, this message translates to:
  /// **'found automatically'**
  String get settingsServerAuto;

  /// No description provided for @settingsFindAuto.
  ///
  /// In en, this message translates to:
  /// **'Find the server automatically'**
  String get settingsFindAuto;

  /// No description provided for @settingsServerSettings.
  ///
  /// In en, this message translates to:
  /// **'Server settings'**
  String get settingsServerSettings;

  /// No description provided for @settingsCatalog.
  ///
  /// In en, this message translates to:
  /// **'Catalog and search'**
  String get settingsCatalog;

  /// No description provided for @settingsCatalogHint.
  ///
  /// In en, this message translates to:
  /// **'TMDB key, search timeout, FlareSolverr'**
  String get settingsCatalogHint;

  /// No description provided for @settingsEngine.
  ///
  /// In en, this message translates to:
  /// **'Torrent engine'**
  String get settingsEngine;

  /// No description provided for @settingsEngineHint.
  ///
  /// In en, this message translates to:
  /// **'Cache, connections, speed limits, protocols'**
  String get settingsEngineHint;

  /// No description provided for @settingsIndexersHint.
  ///
  /// In en, this message translates to:
  /// **'Enable trackers, logins, definitions'**
  String get settingsIndexersHint;

  /// No description provided for @catalogTmdbKey.
  ///
  /// In en, this message translates to:
  /// **'TMDB API key'**
  String get catalogTmdbKey;

  /// No description provided for @catalogTmdbKeyHint.
  ///
  /// In en, this message translates to:
  /// **'v3 API key or v4 token'**
  String get catalogTmdbKeyHint;

  /// No description provided for @catalogTmdbKeySet.
  ///
  /// In en, this message translates to:
  /// **'The key is set. Enter a new one to replace it.'**
  String get catalogTmdbKeySet;

  /// No description provided for @catalogTmdbKeyRemove.
  ///
  /// In en, this message translates to:
  /// **'Remove key'**
  String get catalogTmdbKeyRemove;

  /// No description provided for @catalogTmdbKeyRemoved.
  ///
  /// In en, this message translates to:
  /// **'Key removed'**
  String get catalogTmdbKeyRemoved;

  /// No description provided for @catalogSearchTimeout.
  ///
  /// In en, this message translates to:
  /// **'Tracker search timeout, s'**
  String get catalogSearchTimeout;

  /// No description provided for @catalogFlareSolverr.
  ///
  /// In en, this message translates to:
  /// **'FlareSolverr address'**
  String get catalogFlareSolverr;

  /// No description provided for @catalogFlareSolverrHint.
  ///
  /// In en, this message translates to:
  /// **'For trackers behind a Cloudflare check, e.g. http://192.168.88.1:8191. Empty — off.'**
  String get catalogFlareSolverrHint;

  /// No description provided for @engineCacheSize.
  ///
  /// In en, this message translates to:
  /// **'Cache size, MB'**
  String get engineCacheSize;

  /// No description provided for @engineReadAhead.
  ///
  /// In en, this message translates to:
  /// **'Read-ahead, %'**
  String get engineReadAhead;

  /// No description provided for @enginePreload.
  ///
  /// In en, this message translates to:
  /// **'Preload to disk, % of every file'**
  String get enginePreload;

  /// No description provided for @engineConnections.
  ///
  /// In en, this message translates to:
  /// **'Connection limit'**
  String get engineConnections;

  /// No description provided for @engineDownLimit.
  ///
  /// In en, this message translates to:
  /// **'Download limit, KB/s (0 — unlimited)'**
  String get engineDownLimit;

  /// No description provided for @engineUpLimit.
  ///
  /// In en, this message translates to:
  /// **'Upload limit, KB/s (0 — unlimited)'**
  String get engineUpLimit;

  /// No description provided for @engineDisconnect.
  ///
  /// In en, this message translates to:
  /// **'Unload an idle torrent after, s'**
  String get engineDisconnect;

  /// No description provided for @enginePort.
  ///
  /// In en, this message translates to:
  /// **'Incoming port (0 — random)'**
  String get enginePort;

  /// No description provided for @engineRetrackers.
  ///
  /// In en, this message translates to:
  /// **'Retrackers'**
  String get engineRetrackers;

  /// No description provided for @engineRetrackersNone.
  ///
  /// In en, this message translates to:
  /// **'Leave'**
  String get engineRetrackersNone;

  /// No description provided for @engineRetrackersAdd.
  ///
  /// In en, this message translates to:
  /// **'Add'**
  String get engineRetrackersAdd;

  /// No description provided for @engineRetrackersRemove.
  ///
  /// In en, this message translates to:
  /// **'Remove'**
  String get engineRetrackersRemove;

  /// No description provided for @engineRetrackersReplace.
  ///
  /// In en, this message translates to:
  /// **'Replace'**
  String get engineRetrackersReplace;

  /// No description provided for @engineUseDisk.
  ///
  /// In en, this message translates to:
  /// **'Keep the cache on disk'**
  String get engineUseDisk;

  /// No description provided for @engineSavePath.
  ///
  /// In en, this message translates to:
  /// **'Downloads folder (empty: the data folder)'**
  String get engineSavePath;

  /// No description provided for @engineRemoveCacheOnDrop.
  ///
  /// In en, this message translates to:
  /// **'Delete the cache when a torrent is unloaded'**
  String get engineRemoveCacheOnDrop;

  /// No description provided for @engineUpload.
  ///
  /// In en, this message translates to:
  /// **'Upload (seed)'**
  String get engineUpload;

  /// No description provided for @engineEncrypt.
  ///
  /// In en, this message translates to:
  /// **'Encrypted connections only'**
  String get engineEncrypt;

  /// No description provided for @engineResponsive.
  ///
  /// In en, this message translates to:
  /// **'Responsive mode (read without piece checks)'**
  String get engineResponsive;

  /// No description provided for @engineDebug.
  ///
  /// In en, this message translates to:
  /// **'Debug log'**
  String get engineDebug;

  /// No description provided for @engineReset.
  ///
  /// In en, this message translates to:
  /// **'Reset to defaults'**
  String get engineReset;

  /// No description provided for @engineResetConfirm.
  ///
  /// In en, this message translates to:
  /// **'Reset the torrent engine settings to their defaults?'**
  String get engineResetConfirm;

  /// No description provided for @engineResetDone.
  ///
  /// In en, this message translates to:
  /// **'Settings reset'**
  String get engineResetDone;

  /// No description provided for @engineRestartHint.
  ///
  /// In en, this message translates to:
  /// **'The engine restarts after saving; playing torrents reconnect.'**
  String get engineRestartHint;

  /// No description provided for @indexersSearch.
  ///
  /// In en, this message translates to:
  /// **'Find a tracker'**
  String get indexersSearch;

  /// No description provided for @indexersAll.
  ///
  /// In en, this message translates to:
  /// **'All'**
  String get indexersAll;

  /// No description provided for @indexersEnabled.
  ///
  /// In en, this message translates to:
  /// **'Enabled'**
  String get indexersEnabled;

  /// No description provided for @indexersPublic.
  ///
  /// In en, this message translates to:
  /// **'Public'**
  String get indexersPublic;

  /// No description provided for @indexersPrivate.
  ///
  /// In en, this message translates to:
  /// **'Private'**
  String get indexersPrivate;

  /// No description provided for @indexersTypePublic.
  ///
  /// In en, this message translates to:
  /// **'public'**
  String get indexersTypePublic;

  /// No description provided for @indexersTypeSemi.
  ///
  /// In en, this message translates to:
  /// **'semi-private'**
  String get indexersTypeSemi;

  /// No description provided for @indexersTypePrivate.
  ///
  /// In en, this message translates to:
  /// **'private'**
  String get indexersTypePrivate;

  /// No description provided for @indexersNeedsSetup.
  ///
  /// In en, this message translates to:
  /// **'needs setup'**
  String get indexersNeedsSetup;

  /// No description provided for @indexersError.
  ///
  /// In en, this message translates to:
  /// **'error'**
  String get indexersError;

  /// No description provided for @indexersEmpty.
  ///
  /// In en, this message translates to:
  /// **'No trackers found'**
  String get indexersEmpty;

  /// No description provided for @indexersCounter.
  ///
  /// In en, this message translates to:
  /// **'{enabled} of {total} enabled'**
  String indexersCounter(int enabled, int total);

  /// No description provided for @indexersDefinitions.
  ///
  /// In en, this message translates to:
  /// **'Tracker definitions'**
  String get indexersDefinitions;

  /// No description provided for @indexersDefinitionsInfo.
  ///
  /// In en, this message translates to:
  /// **'Loaded: {count} · updated {date}'**
  String indexersDefinitionsInfo(int count, String date);

  /// No description provided for @indexersDefinitionsNever.
  ///
  /// In en, this message translates to:
  /// **'Loaded: {count} · not updated yet'**
  String indexersDefinitionsNever(int count);

  /// No description provided for @indexersDefinitionsUpdate.
  ///
  /// In en, this message translates to:
  /// **'Update definitions'**
  String get indexersDefinitionsUpdate;

  /// No description provided for @indexersDefinitionsUpdating.
  ///
  /// In en, this message translates to:
  /// **'Definitions are downloading…'**
  String get indexersDefinitionsUpdating;

  /// No description provided for @indexersDefinitionsUpdated.
  ///
  /// In en, this message translates to:
  /// **'Definitions updated: {count}'**
  String indexersDefinitionsUpdated(int count);

  /// No description provided for @indexersDefinitionsFailed.
  ///
  /// In en, this message translates to:
  /// **'The last update failed: {error}'**
  String indexersDefinitionsFailed(String error);

  /// No description provided for @indexersDefinitionsEmpty.
  ///
  /// In en, this message translates to:
  /// **'The server has no tracker definitions yet. Press “Update definitions” to download them.'**
  String get indexersDefinitionsEmpty;

  /// No description provided for @indexerEnabled.
  ///
  /// In en, this message translates to:
  /// **'Tracker enabled'**
  String get indexerEnabled;

  /// No description provided for @indexerSiteLink.
  ///
  /// In en, this message translates to:
  /// **'Site address'**
  String get indexerSiteLink;

  /// No description provided for @indexerCustomLink.
  ///
  /// In en, this message translates to:
  /// **'Custom address'**
  String get indexerCustomLink;

  /// No description provided for @indexerSettings.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get indexerSettings;

  /// No description provided for @indexerNoSettings.
  ///
  /// In en, this message translates to:
  /// **'This tracker has no settings'**
  String get indexerNoSettings;

  /// No description provided for @indexerPasswordSet.
  ///
  /// In en, this message translates to:
  /// **'set — leave empty to keep it'**
  String get indexerPasswordSet;

  /// No description provided for @indexerTest.
  ///
  /// In en, this message translates to:
  /// **'Test'**
  String get indexerTest;

  /// No description provided for @indexerTestOk.
  ///
  /// In en, this message translates to:
  /// **'Works: {results} results in {ms} ms'**
  String indexerTestOk(int results, int ms);

  /// No description provided for @indexerTestFail.
  ///
  /// In en, this message translates to:
  /// **'Error: {error}'**
  String indexerTestFail(String error);

  /// No description provided for @indexerLastError.
  ///
  /// In en, this message translates to:
  /// **'Last error'**
  String get indexerLastError;

  /// No description provided for @indexerSaved.
  ///
  /// In en, this message translates to:
  /// **'Tracker settings saved'**
  String get indexerSaved;

  /// No description provided for @settingsVideoOutput.
  ///
  /// In en, this message translates to:
  /// **'Video output'**
  String get settingsVideoOutput;

  /// No description provided for @videoOutputHardware.
  ///
  /// In en, this message translates to:
  /// **'Hardware — MediaCodec (HDR, faster)'**
  String get videoOutputHardware;

  /// No description provided for @videoOutputHardwareHint.
  ///
  /// In en, this message translates to:
  /// **'Decodes straight to the screen: 10-bit, HDR10, HDR10+ and Dolby Vision'**
  String get videoOutputHardwareHint;

  /// No description provided for @videoOutputCompatible.
  ///
  /// In en, this message translates to:
  /// **'Compatible — OpenGL'**
  String get videoOutputCompatible;

  /// No description provided for @videoOutputCompatibleHint.
  ///
  /// In en, this message translates to:
  /// **'Try it if the picture is black, glitchy or does not start'**
  String get videoOutputCompatibleHint;

  /// No description provided for @playerVideoOutputHint.
  ///
  /// In en, this message translates to:
  /// **'Playback problem? Try Settings → Video output → Compatible — OpenGL'**
  String get playerVideoOutputHint;

  /// No description provided for @watchTorrents.
  ///
  /// In en, this message translates to:
  /// **'Torrents'**
  String get watchTorrents;

  /// No description provided for @watchTorrentsHint.
  ///
  /// In en, this message translates to:
  /// **'Find a release on the trackers'**
  String get watchTorrentsHint;

  /// No description provided for @watchTrailers.
  ///
  /// In en, this message translates to:
  /// **'Trailers'**
  String get watchTrailers;

  /// No description provided for @watchTrailersHint.
  ///
  /// In en, this message translates to:
  /// **'Trailers and videos on YouTube'**
  String get watchTrailersHint;

  /// No description provided for @trailersTitle.
  ///
  /// In en, this message translates to:
  /// **'Trailers'**
  String get trailersTitle;

  /// No description provided for @trailersLoading.
  ///
  /// In en, this message translates to:
  /// **'Loading videos…'**
  String get trailersLoading;

  /// No description provided for @trailersEmpty.
  ///
  /// In en, this message translates to:
  /// **'No trailers'**
  String get trailersEmpty;

  /// No description provided for @trailersOpenFailed.
  ///
  /// In en, this message translates to:
  /// **'Could not open YouTube: {error}'**
  String trailersOpenFailed(String error);

  /// No description provided for @preload.
  ///
  /// In en, this message translates to:
  /// **'Preload'**
  String get preload;

  /// No description provided for @preloadStarted.
  ///
  /// In en, this message translates to:
  /// **'Preload started: {name}'**
  String preloadStarted(String name);

  /// No description provided for @preloadProgress.
  ///
  /// In en, this message translates to:
  /// **'Preload: {done} / {total}'**
  String preloadProgress(String done, String total);

  /// No description provided for @downloadAll.
  ///
  /// In en, this message translates to:
  /// **'Download in full'**
  String get downloadAll;

  /// No description provided for @downloadAllCancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel full download'**
  String get downloadAllCancel;

  /// No description provided for @downloadAllStarted.
  ///
  /// In en, this message translates to:
  /// **'Full download started — the torrent stays on the server\'s disk'**
  String get downloadAllStarted;

  /// No description provided for @downloadAllCancelled.
  ///
  /// In en, this message translates to:
  /// **'Full download cancelled'**
  String get downloadAllCancelled;

  /// No description provided for @downloadAllNoRoom.
  ///
  /// In en, this message translates to:
  /// **'The torrent does not fit: the server disk lacks free space'**
  String get downloadAllNoRoom;

  /// No description provided for @downloadAllProgress.
  ///
  /// In en, this message translates to:
  /// **'In full: {done} / {total}'**
  String downloadAllProgress(String done, String total);

  /// No description provided for @downloadAllDone.
  ///
  /// In en, this message translates to:
  /// **'Downloaded in full: {total}'**
  String downloadAllDone(String total);

  /// No description provided for @navTorrents.
  ///
  /// In en, this message translates to:
  /// **'TORRENTS'**
  String get navTorrents;

  /// No description provided for @torrentsTitle.
  ///
  /// In en, this message translates to:
  /// **'Torrents'**
  String get torrentsTitle;

  /// No description provided for @torrentsAdd.
  ///
  /// In en, this message translates to:
  /// **'Add'**
  String get torrentsAdd;

  /// No description provided for @torrentsAddTitle.
  ///
  /// In en, this message translates to:
  /// **'Add a torrent'**
  String get torrentsAddTitle;

  /// No description provided for @torrentsLink.
  ///
  /// In en, this message translates to:
  /// **'Magnet, .torrent link or hash'**
  String get torrentsLink;

  /// No description provided for @torrentsTitleField.
  ///
  /// In en, this message translates to:
  /// **'Title (optional)'**
  String get torrentsTitleField;

  /// No description provided for @torrentsAdded.
  ///
  /// In en, this message translates to:
  /// **'Torrent added'**
  String get torrentsAdded;

  /// No description provided for @torrentsEmpty.
  ///
  /// In en, this message translates to:
  /// **'No torrents. Add a release from search or by a magnet link.'**
  String get torrentsEmpty;

  /// No description provided for @torrentsDrop.
  ///
  /// In en, this message translates to:
  /// **'Unload from memory'**
  String get torrentsDrop;

  /// No description provided for @torrentsDropped.
  ///
  /// In en, this message translates to:
  /// **'Unloaded from memory'**
  String get torrentsDropped;

  /// No description provided for @torrentsDelete.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get torrentsDelete;

  /// No description provided for @torrentsDeleteConfirm.
  ///
  /// In en, this message translates to:
  /// **'Delete “{title}” from the torrent list?'**
  String torrentsDeleteConfirm(String title);

  /// No description provided for @torrentsDeleted.
  ///
  /// In en, this message translates to:
  /// **'Torrent deleted'**
  String get torrentsDeleted;

  /// No description provided for @torrentsDetails.
  ///
  /// In en, this message translates to:
  /// **'Details'**
  String get torrentsDetails;

  /// No description provided for @torrentsStatus.
  ///
  /// In en, this message translates to:
  /// **'Status'**
  String get torrentsStatus;

  /// No description provided for @torrentsSize.
  ///
  /// In en, this message translates to:
  /// **'Size'**
  String get torrentsSize;

  /// No description provided for @torrentsLoaded.
  ///
  /// In en, this message translates to:
  /// **'Loaded'**
  String get torrentsLoaded;

  /// No description provided for @torrentsSpeedDown.
  ///
  /// In en, this message translates to:
  /// **'Download'**
  String get torrentsSpeedDown;

  /// No description provided for @torrentsSpeedUp.
  ///
  /// In en, this message translates to:
  /// **'Upload'**
  String get torrentsSpeedUp;

  /// No description provided for @torrentsPeers.
  ///
  /// In en, this message translates to:
  /// **'Peers'**
  String get torrentsPeers;

  /// No description provided for @torrentsSeeders.
  ///
  /// In en, this message translates to:
  /// **'Seeders'**
  String get torrentsSeeders;

  /// No description provided for @torrentsFull.
  ///
  /// In en, this message translates to:
  /// **'Full'**
  String get torrentsFull;

  /// No description provided for @torrentsFullDone.
  ///
  /// In en, this message translates to:
  /// **'Downloaded in full'**
  String get torrentsFullDone;

  /// No description provided for @torrentsHash.
  ///
  /// In en, this message translates to:
  /// **'Hash'**
  String get torrentsHash;

  /// No description provided for @torrentsHashCopied.
  ///
  /// In en, this message translates to:
  /// **'Hash copied'**
  String get torrentsHashCopied;

  /// No description provided for @torrentsAddedAt.
  ///
  /// In en, this message translates to:
  /// **'Added'**
  String get torrentsAddedAt;

  /// No description provided for @torrentsCache.
  ///
  /// In en, this message translates to:
  /// **'Cache'**
  String get torrentsCache;

  /// No description provided for @torrentsCacheSummary.
  ///
  /// In en, this message translates to:
  /// **'Filled {filled} / {capacity} · pieces: {count} · piece size {pieceSize}'**
  String torrentsCacheSummary(
    String filled,
    String capacity,
    int count,
    String pieceSize,
  );

  /// No description provided for @torrentsNoCache.
  ///
  /// In en, this message translates to:
  /// **'The cache is empty — the torrent is not active'**
  String get torrentsNoCache;

  /// No description provided for @torrentsLegendDone.
  ///
  /// In en, this message translates to:
  /// **'loaded'**
  String get torrentsLegendDone;

  /// No description provided for @torrentsLegendPartial.
  ///
  /// In en, this message translates to:
  /// **'partial'**
  String get torrentsLegendPartial;

  /// No description provided for @torrentsLegendQueued.
  ///
  /// In en, this message translates to:
  /// **'queued'**
  String get torrentsLegendQueued;

  /// No description provided for @torrentsLegendReader.
  ///
  /// In en, this message translates to:
  /// **'reading'**
  String get torrentsLegendReader;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'ru'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'ru':
      return AppLocalizationsRu();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
