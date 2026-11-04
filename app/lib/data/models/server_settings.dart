/// Server settings managed by the app: /api/config, /api/indexers,
/// /api/definitions and /api/engine/settings (see docs/en/31-rest-api.md).
library;

/// GET/PUT /api/config.
class ServerConfig {
  /// A key is set (the server never returns the key itself).
  final bool tmdbApiKeySet;
  final int searchTimeoutSec;
  final String flareSolverrUrl;

  const ServerConfig({required this.tmdbApiKeySet, required this.searchTimeoutSec, required this.flareSolverrUrl});

  factory ServerConfig.fromJson(Map<String, dynamic>? j) => ServerConfig(
    tmdbApiKeySet: j?['tmdbApiKeySet'] == true,
    searchTimeoutSec: (j?['searchTimeoutSec'] as num?)?.toInt() ?? 30,
    flareSolverrUrl: j?['flareSolverrUrl'] as String? ?? '',
  );
}

/// A GET /api/indexers item.
class Indexer {
  final String id;
  final String name;
  final String description;
  final String language;

  /// public | semi-private | private.
  final String type;
  final List<String> links;
  final bool enabled;

  /// All required settings are filled in.
  final bool configured;
  final String lastError;
  final DateTime? lastCheckedAt;

  const Indexer({
    required this.id,
    required this.name,
    this.description = '',
    this.language = '',
    this.type = 'public',
    this.links = const [],
    this.enabled = false,
    this.configured = true,
    this.lastError = '',
    this.lastCheckedAt,
  });

  String get title => name.isEmpty ? id : name;
  bool get isPublic => type == 'public';

  factory Indexer.fromJson(Map<String, dynamic> j) => Indexer(
    id: j['id'] as String? ?? '',
    name: j['name'] as String? ?? '',
    description: j['description'] as String? ?? '',
    language: j['language'] as String? ?? '',
    type: j['type'] as String? ?? 'public',
    links: [for (final l in (j['links'] as List? ?? const [])) '$l'],
    enabled: j['enabled'] == true,
    configured: j['configured'] != false,
    lastError: j['lastError'] as String? ?? '',
    lastCheckedAt: DateTime.tryParse(j['lastCheckedAt'] as String? ?? ''),
  );

  Indexer copyWith({bool? enabled}) => Indexer(
    id: id,
    name: name,
    description: description,
    language: language,
    type: type,
    links: links,
    enabled: enabled ?? this.enabled,
    configured: configured,
    lastError: lastError,
    lastCheckedAt: lastCheckedAt,
  );
}

class SettingOption {
  final String value;
  final String label;
  const SettingOption(this.value, this.label);
}

/// A tracker form field (Cardigann setting).
class IndexerSetting {
  final String name;

  /// text | password | checkbox | select | multi-select | info*.
  final String type;
  final String label;
  final Object? defaultValue;
  final List<SettingOption> options;

  const IndexerSetting({
    required this.name,
    required this.type,
    this.label = '',
    this.defaultValue,
    this.options = const [],
  });

  factory IndexerSetting.fromJson(Map<String, dynamic> j) => IndexerSetting(
    name: j['name'] as String? ?? '',
    type: j['type'] as String? ?? 'text',
    label: j['label'] as String? ?? '',
    defaultValue: j['default'],
    options: [
      for (final o in (j['options'] as List? ?? const []))
        SettingOption('${(o as Map)['value'] ?? ''}', '${o['label'] ?? o['value'] ?? ''}'),
    ],
  );

  bool get isInfo => type.startsWith('info');
  String get title => label.isEmpty ? name : label;

  static bool _truthy(Object? v) => v == true || v == 'true' || v == '1' || v == 'on';

  static List<String> _list(Object? v) {
    if (v is List) return [for (final e in v) '$e'];
    if (v is String) {
      return [
        for (final s in v.split(','))
          if (s.trim().isNotEmpty) s.trim(),
      ];
    }
    return const [];
  }

  /// Saved value (or default) → form value:
  /// bool for checkbox, `List<String>` for multi-select, otherwise a string.
  /// A password always starts empty: an empty string keeps the saved one.
  Object formValue(Object? saved) {
    final raw = saved ?? defaultValue;
    switch (type) {
      case 'checkbox':
        return _truthy(raw);
      case 'multi-select':
        return _list(raw);
      case 'password':
        return saved is String ? saved : '';
      default:
        return raw == null ? '' : '$raw';
    }
  }

  /// Form value → PUT value in the same shape the server sent it.
  Object wireValue(Object value, Object? saved) {
    final sample = saved ?? defaultValue;
    if (type == 'checkbox') {
      final b = value == true;
      return sample is String ? '$b' : b;
    }
    if (type == 'multi-select') {
      final list = value is List ? [for (final e in value) '$e'] : _list(value);
      return sample is String ? list.join(',') : list;
    }
    return value is String ? value : '$value';
  }
}

/// GET /api/indexers/{id}.
class IndexerDetail {
  final Indexer indexer;
  final List<IndexerSetting> settings;
  final Map<String, Object?> config;

  const IndexerDetail({required this.indexer, this.settings = const [], this.config = const {}});

  factory IndexerDetail.fromJson(Map<String, dynamic> j) => IndexerDetail(
    indexer: Indexer.fromJson(j),
    settings: [
      for (final s in (j['settings'] as List? ?? const [])) IndexerSetting.fromJson(s as Map<String, dynamic>),
    ],
    config: Map<String, Object?>.from(j['config'] as Map? ?? const {}),
  );

  /// Form fields excluding the site address picker (it is separate).
  List<IndexerSetting> get fields => [
    for (final s in settings)
      if (s.name != 'sitelink') s,
  ];

  bool passwordSet(String name) => config['${name}__set'] == '1';

  String get siteLink {
    final v = config['sitelink'];
    if (v is String && v.isNotEmpty) return v;
    return indexer.links.isEmpty ? '' : indexer.links.first;
  }
}

/// POST /api/indexers/{id}/test.
class IndexerTestResult {
  final bool ok;
  final int results;
  final String error;
  final int elapsedMs;

  const IndexerTestResult({required this.ok, this.results = 0, this.error = '', this.elapsedMs = 0});

  factory IndexerTestResult.fromJson(Map<String, dynamic>? j) => IndexerTestResult(
    ok: j?['ok'] == true,
    results: (j?['results'] as num?)?.toInt() ?? 0,
    error: j?['error'] as String? ?? '',
    elapsedMs: (j?['elapsedMs'] as num?)?.toInt() ?? 0,
  );
}

/// GET /api/definitions.
class DefinitionsStatus {
  final int count;
  final DateTime? updatedAt;
  final bool updating;
  final String lastError;

  const DefinitionsStatus({this.count = 0, this.updatedAt, this.updating = false, this.lastError = ''});

  factory DefinitionsStatus.fromJson(Map<String, dynamic>? j) => DefinitionsStatus(
    count: (j?['count'] as num?)?.toInt() ?? 0,
    updatedAt: DateTime.tryParse(j?['updatedAt'] as String? ?? ''),
    updating: j?['updating'] == true,
    lastError: j?['lastError'] as String? ?? '',
  );
}

/// GET/PUT /api/engine/settings. PUT takes the whole object, so the original JSON is kept
/// as is and only known fields are changed — new server fields are not lost.
class EngineSettings {
  final Map<String, Object?> json;
  const EngineSettings(this.json);

  factory EngineSettings.fromJson(Map<String, dynamic>? j) => EngineSettings(Map<String, Object?>.from(j ?? const {}));

  int intOf(String key) => (json[key] as num?)?.toInt() ?? 0;
  bool boolOf(String key) => json[key] == true;

  EngineSettings copyWith(String key, Object value) => EngineSettings({...json, key: value});
}
