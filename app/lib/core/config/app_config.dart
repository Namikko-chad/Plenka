/// Build-time constants. Overridden via --dart-define.
class AppConfig {
  /// plenka-server port.
  static const int serverPort = int.fromEnvironment('SERVER_PORT', defaultValue: 8095);

  /// Explicit server address, e.g. http://192.168.88.1:8095. Empty — discover automatically.
  static const String serverUrl = String.fromEnvironment('SERVER_URL');
}
