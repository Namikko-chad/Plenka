import 'package:flutter/material.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../data/api/config_api.dart';
import '../../data/models/server_settings.dart';
import '../../l10n/l10n.dart';
import '../common/widgets/common.dart';
import 'form.dart';

/// Catalog and search: TMDB key, search timeout, FlareSolverr. Catalog language = UI language.
class CatalogSettingsScreen extends StatefulWidget {
  const CatalogSettingsScreen({super.key});

  @override
  State<CatalogSettingsScreen> createState() => _CatalogSettingsScreenState();
}

class _CatalogSettingsScreenState extends State<CatalogSettingsScreen> {
  final _api = getIt<ConfigApi>();
  final _key = TextEditingController();
  final _timeout = TextEditingController();
  final _flare = TextEditingController();
  ServerConfig? _config;
  Object? _error;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _key.dispose();
    _timeout.dispose();
    _flare.dispose();
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

  void _apply(ServerConfig c) {
    if (!mounted) return;
    setState(() {
      _config = c;
      _key.clear();
      _timeout.text = '${c.searchTimeoutSec}';
      _flare.text = c.flareSolverrUrl;
    });
  }

  Future<void> _save() async {
    final l = context.l10n;
    setState(() => _saving = true);
    try {
      _apply(
        await _api.put({
          'tmdbApiKey': _key.text.trim(),
          'searchTimeoutSec': int.tryParse(_timeout.text) ?? _config!.searchTimeoutSec,
          'flareSolverrUrl': _flare.text.trim(),
        }),
      );
      if (mounted) showMessage(context, l.saved);
    } catch (e) {
      if (mounted) showMessage(context, errorText(e));
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _removeKey() async {
    final l = context.l10n;
    if (!await confirmDialog(context, title: l.catalogTmdbKeyRemove, action: l.catalogTmdbKeyRemove)) return;
    try {
      _apply(await _api.put({'tmdbApiKey': '-'}));
      if (mounted) showMessage(context, l.catalogTmdbKeyRemoved);
    } catch (e) {
      if (mounted) showMessage(context, errorText(e));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final c = _config;
    return SettingsPage(
      title: l.settingsCatalog,
      loading: c == null,
      error: _error,
      onRetry: _load,
      children: () => [
        const SizedBox(height: 12),
        LabeledField(
          label: l.catalogTmdbKey,
          controller: _key,
          autofocus: true,
          hint: c!.tmdbApiKeySet ? '••••••••' : l.catalogTmdbKeyHint,
          helper: c.tmdbApiKeySet ? l.catalogTmdbKeySet : null,
        ),
        if (c.tmdbApiKeySet)
          Align(
            alignment: Alignment.centerLeft,
            child: Padding(
              padding: const EdgeInsets.only(bottom: 16),
              child: OutlinedButton.icon(
                style: AppButtons.of(ButtonVariant.red),
                onPressed: _removeKey,
                icon: const Icon(Icons.delete_outline, size: 16),
                label: Text(l.catalogTmdbKeyRemove),
              ),
            ),
          ),
        LabeledField(label: l.catalogSearchTimeout, controller: _timeout, number: true),
        LabeledField(
          label: l.catalogFlareSolverr,
          controller: _flare,
          keyboardType: TextInputType.url,
          hint: 'http://',
          helper: l.catalogFlareSolverrHint,
        ),
        Align(
          alignment: Alignment.centerLeft,
          child: BusyButton(label: l.save, icon: Icons.save_outlined, busy: _saving, onPressed: _save),
        ),
      ],
    );
  }
}
