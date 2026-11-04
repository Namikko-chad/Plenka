import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

enum PlayerMode { internal, external, ask }

/// Video output of the built-in player on Android.
/// [hardware] — MediaCodec renders into a Surface (10-bit, HDR10/HDR10+/Dolby Vision, faster);
/// [compatible] — media_kit defaults (OpenGL, frames copied into a texture).
enum VideoOutput { hardware, compatible }

/// UI language. [system] — the device language (English if unsupported).
enum AppLanguage {
  system(null),
  en('en'),
  ru('ru');

  final String? code;
  const AppLanguage(this.code);
}

class SettingsStore {
  final SharedPreferences _prefs;

  SettingsStore(this._prefs)
    : language = ValueNotifier(AppLanguage.values.asNameMap()[_prefs.getString(_kLanguage)] ?? AppLanguage.system);

  static const _kServer = 'server_url';
  static const _kManual = 'server_manual';
  static const _kPlayer = 'player_mode';
  static const _kLanguage = 'app_language';
  static const _kVideoOutput = 'video_output';

  String? get serverUrl => _prefs.getString(_kServer);
  Future<void> setServerUrl(String? v) => v == null ? _prefs.remove(_kServer) : _prefs.setString(_kServer, v);

  /// The address was set manually: on launch connect only to it; auto-discovery doesn't replace it.
  bool get serverManual => _prefs.getBool(_kManual) ?? false;
  Future<void> setServerManual(bool v) => _prefs.setBool(_kManual, v);

  PlayerMode get playerMode => PlayerMode.values.asNameMap()[_prefs.getString(_kPlayer)] ?? PlayerMode.internal;
  Future<void> setPlayerMode(PlayerMode m) => _prefs.setString(_kPlayer, m.name);

  VideoOutput get videoOutput =>
      VideoOutput.values.asNameMap()[_prefs.getString(_kVideoOutput)] ?? VideoOutput.hardware;
  Future<void> setVideoOutput(VideoOutput v) => _prefs.setString(_kVideoOutput, v.name);

  /// The selected language; the app listens to it and rebuilds.
  final ValueNotifier<AppLanguage> language;
  Future<void> setLanguage(AppLanguage l) async {
    language.value = l;
    await _prefs.setString(_kLanguage, l.name);
  }
}
