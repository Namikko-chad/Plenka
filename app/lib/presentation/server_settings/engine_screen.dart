import 'package:flutter/material.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../data/api/engine_api.dart';
import '../../data/models/server_settings.dart';
import '../../l10n/l10n.dart';
import '../common/widgets/common.dart';
import 'form.dart';

const _mb = 1024 * 1024;

/// Numeric engine fields: JSON key, label, allowed range.
typedef _NumField = (String key, String Function(AppLocalizations) label, int min, int max);

final List<_NumField> _numbers = [
  ('cacheSize', (l) => l.engineCacheSize, 0, 1 << 20), // in MB, see _toForm/_fromForm
  ('readerReadAhead', (l) => l.engineReadAhead, 5, 100),
  ('preloadCache', (l) => l.enginePreload, 0, 100),
  ('connectionsLimit', (l) => l.engineConnections, 0, 1 << 16),
  ('downloadRateLimit', (l) => l.engineDownLimit, 0, 1 << 30),
  ('uploadRateLimit', (l) => l.engineUpLimit, 0, 1 << 30),
  ('torrentDisconnectTimeout', (l) => l.engineDisconnect, 0, 1 << 20),
  ('peersListenPort', (l) => l.enginePort, 0, 65535),
];

/// Switches phrased as "enabled" on top of the engine's disable* flags.
typedef _Toggle = (String key, String Function(AppLocalizations) label, bool inverted);

final List<_Toggle> _toggles = [
  ('disableDHT', (_) => 'DHT', true),
  ('disablePEX', (_) => 'PEX', true),
  ('disableUPNP', (_) => 'UPnP', true),
  ('disableTCP', (_) => 'TCP', true),
  ('disableUTP', (_) => 'uTP', true),
  ('enableIPv6', (_) => 'IPv6', false),
  ('disableUpload', (l) => l.engineUpload, true),
  ('forceEncrypt', (l) => l.engineEncrypt, false),
  ('responsiveMode', (l) => l.engineResponsive, false),
  ('enableDebug', (l) => l.engineDebug, false),
];

class EngineSettingsScreen extends StatefulWidget {
  const EngineSettingsScreen({super.key});

  @override
  State<EngineSettingsScreen> createState() => _EngineSettingsScreenState();
}

class _EngineSettingsScreenState extends State<EngineSettingsScreen> {
  final _api = getIt<EngineApi>();
  final _fields = {for (final f in _numbers) f.$1: TextEditingController()};
  final _savePath = TextEditingController();
  EngineSettings? _settings;
  Object? _error;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    for (final c in _fields.values) {
      c.dispose();
    }
    _savePath.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      _apply(await _api.get());
    } catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  void _apply(EngineSettings s) {
    if (!mounted) return;
    setState(() {
      _settings = s;
      for (final (key, _, _, _) in _numbers) {
        final v = s.intOf(key);
        _fields[key]!.text = '${key == 'cacheSize' ? (v / _mb).round() : v}';
      }
      _savePath.text = (s.json['torrentsSavePath'] as String?) ?? '';
    });
  }

  /// Form values → the full object for PUT (fields outside the form stay as received).
  EngineSettings _collect() {
    var s = _settings!;
    for (final (key, _, min, max) in _numbers) {
      final v = (int.tryParse(_fields[key]!.text) ?? s.intOf(key)).clamp(min, max);
      s = s.copyWith(key, key == 'cacheSize' ? v * _mb : v);
    }
    return s.copyWith('torrentsSavePath', _savePath.text.trim());
  }

  Future<void> _save() async {
    final l = context.l10n;
    setState(() => _saving = true);
    try {
      _apply(await _api.save(_collect()));
      if (mounted) showMessage(context, l.saved);
    } catch (e) {
      if (mounted) showMessage(context, errorText(e));
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _reset() async {
    final l = context.l10n;
    if (!await confirmDialog(context, title: l.engineReset, text: l.engineResetConfirm, action: l.engineReset)) return;
    try {
      _apply(await _api.reset());
      if (mounted) showMessage(context, l.engineResetDone);
    } catch (e) {
      if (mounted) showMessage(context, errorText(e));
    }
  }

  void _set(String key, Object value) => setState(() => _settings = _collect().copyWith(key, value));

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final s = _settings;
    return SettingsPage(
      title: l.settingsEngine,
      loading: s == null,
      error: _error,
      onRetry: _load,
      children: () => [
        Padding(
          padding: const EdgeInsets.only(top: 8, bottom: 16),
          child: Text(l.engineRestartHint, style: AppText.small),
        ),
        for (final (i, (key, label, _, _)) in _numbers.indexed)
          LabeledField(label: label(l), controller: _fields[key]!, number: true, autofocus: i == 0),
        FieldLabel(l.engineRetrackers),
        Align(
          alignment: Alignment.centerLeft,
          child: Segmented<int>(
            semanticLabel: l.engineRetrackers,
            options: [
              (0, l.engineRetrackersNone),
              (1, l.engineRetrackersAdd),
              (2, l.engineRetrackersRemove),
              (3, l.engineRetrackersReplace),
            ],
            value: s!.intOf('retrackersMode'),
            onChanged: (v) => _set('retrackersMode', v),
          ),
        ),
        const SizedBox(height: 16),
        LabeledField(label: l.engineSavePath, controller: _savePath),
        const SizedBox(height: 8),
        SwitchRow(label: l.engineUseDisk, value: s.boolOf('useDisk'), onChanged: (v) => _set('useDisk', v)),
        if (s.boolOf('useDisk')) ...[
          SwitchRow(
            label: l.engineRemoveCacheOnDrop,
            value: s.boolOf('removeCacheOnDrop'),
            onChanged: (v) => _set('removeCacheOnDrop', v),
          ),
        ],
        for (final (key, label, inverted) in _toggles)
          SwitchRow(label: label(l), value: s.boolOf(key) != inverted, onChanged: (v) => _set(key, v != inverted)),
        const SizedBox(height: 20),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            BusyButton(label: l.save, icon: Icons.save_outlined, busy: _saving, onPressed: _save),
            OutlinedButton.icon(
              style: AppButtons.of(ButtonVariant.red),
              onPressed: _saving ? null : _reset,
              icon: const Icon(Icons.restart_alt, size: 16),
              label: Text(l.engineReset),
            ),
          ],
        ),
      ],
    );
  }
}
