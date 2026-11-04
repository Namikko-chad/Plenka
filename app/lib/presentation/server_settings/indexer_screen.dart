import 'package:flutter/material.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../../core/utils/format.dart';
import '../../data/api/indexers_api.dart';
import '../../data/models/server_settings.dart';
import '../../l10n/l10n.dart';
import '../common/widgets/common.dart';
import 'form.dart';

/// The "custom address" value in the site address picker.
const _custom = '\u0000custom';

/// Settings of a single tracker: enabling, site address, definition fields (login, password,
/// checkboxes, lists), test.
class IndexerScreen extends StatefulWidget {
  final String id;
  const IndexerScreen({super.key, required this.id});

  @override
  State<IndexerScreen> createState() => _IndexerScreenState();
}

class _IndexerScreenState extends State<IndexerScreen> {
  final _api = getIt<IndexersApi>();
  IndexerDetail? _detail;
  Object? _error;
  bool _enabled = false;
  String _siteLink = '';
  final _customLink = TextEditingController();

  /// Values of checkboxes, lists and dropdowns.
  final _values = <String, Object>{};

  /// Text and password fields.
  final _text = <String, TextEditingController>{};
  bool _saving = false;
  bool _testing = false;
  IndexerTestResult? _test;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _customLink.dispose();
    for (final c in _text.values) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      _reset(await _api.get(widget.id));
    } catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  static bool _isText(IndexerSetting s) => !const {'checkbox', 'select', 'multi-select'}.contains(s.type);

  void _reset(IndexerDetail d) {
    if (!mounted) return;
    setState(() {
      _detail = d;
      _enabled = d.indexer.enabled;
      final link = d.siteLink;
      if (link.isEmpty || d.indexer.links.contains(link)) {
        _siteLink = link;
        _customLink.clear();
      } else {
        _siteLink = _custom;
        _customLink.text = link;
      }
      _values.clear();
      for (final s in d.fields) {
        if (s.isInfo) continue;
        final v = s.formValue(d.config[s.name]);
        if (_isText(s)) {
          (_text[s.name] ??= TextEditingController()).text = v as String;
        } else {
          _values[s.name] = v;
        }
      }
    });
  }

  /// Saves the form; false — the server rejected it (the message is already shown).
  Future<bool> _persist() async {
    final d = _detail!;
    final config = <String, Object?>{};
    for (final s in d.fields) {
      if (s.isInfo) continue;
      final value = _isText(s) ? _text[s.name]!.text : _values[s.name]!;
      config[s.name] = s.wireValue(value, d.config[s.name]);
    }
    final link = _siteLink == _custom ? _customLink.text.trim() : _siteLink;
    if (link.isNotEmpty) config['sitelink'] = link;
    try {
      _reset(await _api.save(d.indexer.id, enabled: _enabled, config: config));
      return true;
    } catch (e) {
      if (mounted) showMessage(context, errorText(e));
      return false;
    }
  }

  Future<void> _save() async {
    final l = context.l10n;
    setState(() => _saving = true);
    if (await _persist() && mounted) showMessage(context, l.indexerSaved);
    if (mounted) setState(() => _saving = false);
  }

  /// Tests what is currently in the form: saves it first.
  Future<void> _runTest() async {
    setState(() {
      _testing = true;
      _test = null;
    });
    try {
      if (!await _persist()) return;
      final r = await _api.test(widget.id);
      if (mounted) setState(() => _test = r);
      // the server's "last error" got updated by the test
      final d = await _api.get(widget.id);
      if (mounted) setState(() => _detail = d);
    } catch (e) {
      if (mounted) showMessage(context, errorText(e));
    } finally {
      if (mounted) setState(() => _testing = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final d = _detail;
    return SettingsPage(
      title: d?.indexer.title ?? widget.id,
      loading: d == null,
      error: _error,
      onRetry: _load,
      children: () => [
        if (d!.indexer.description.isNotEmpty)
          Padding(
            padding: const EdgeInsets.only(top: 8, bottom: 8),
            child: Text(d.indexer.description, style: AppText.small),
          ),
        SwitchRow(
          label: l.indexerEnabled,
          value: _enabled,
          autofocus: true,
          onChanged: (v) => setState(() => _enabled = v),
        ),
        if (d.indexer.lastError.isNotEmpty) ...[
          const SizedBox(height: 8),
          AppBanner(
            tone: Tone.error,
            icon: Icons.error_outline,
            message: [
              '${l.indexerLastError}: ${d.indexer.lastError}',
              if (d.indexer.lastCheckedAt != null) formatDateTime(d.indexer.lastCheckedAt!),
            ].join('\n'),
          ),
        ],
        if (d.indexer.links.isNotEmpty) ...[
          SectionTitle(l.indexerSiteLink, inset: false),
          RadioGroup<String>(
            groupValue: _siteLink,
            onChanged: (v) => setState(() => _siteLink = v ?? _siteLink),
            child: Column(
              children: [
                for (final link in d.indexer.links)
                  RadioListTile<String>(
                    value: link,
                    title: Text(link, style: AppText.body),
                  ),
                RadioListTile<String>(
                  value: _custom,
                  title: Text(l.indexerCustomLink, style: AppText.body),
                ),
              ],
            ),
          ),
          if (_siteLink == _custom)
            LabeledField(
              label: l.indexerCustomLink,
              controller: _customLink,
              hint: 'https://',
              keyboardType: TextInputType.url,
            ),
        ],
        SectionTitle(l.indexerSettings, inset: false),
        if (d.fields.isEmpty) Text(l.indexerNoSettings, style: AppText.small),
        for (final s in d.fields) _field(s, d),
        const SizedBox(height: 20),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            BusyButton(label: l.save, icon: Icons.save_outlined, busy: _saving, onPressed: _testing ? null : _save),
            BusyButton(
              label: l.indexerTest,
              icon: Icons.network_check,
              busy: _testing,
              variant: ButtonVariant.cyan,
              onPressed: _saving ? null : _runTest,
            ),
          ],
        ),
        if (_test != null) ...[
          const SizedBox(height: 16),
          AppBanner(
            tone: _test!.ok ? Tone.ok : Tone.error,
            icon: _test!.ok ? Icons.check_circle_outline : Icons.error_outline,
            message: _test!.ok ? l.indexerTestOk(_test!.results, _test!.elapsedMs) : l.indexerTestFail(_test!.error),
          ),
        ],
      ],
    );
  }

  Widget _field(IndexerSetting s, IndexerDetail d) {
    final l = context.l10n;
    if (s.isInfo) {
      final text = htmlToText('${s.defaultValue ?? ''}');
      if (text.isEmpty && s.label.isEmpty) return const SizedBox.shrink();
      return Padding(
        padding: const EdgeInsets.only(bottom: 16),
        child: AppCard(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (s.label.isNotEmpty) FieldLabel(s.label),
              Text(text, style: AppText.small.copyWith(color: AppColors.text, height: 1.45)),
            ],
          ),
        ),
      );
    }
    switch (s.type) {
      case 'checkbox':
        return SwitchRow(
          label: s.title,
          value: _values[s.name] == true,
          onChanged: (v) => setState(() => _values[s.name] = v),
        );
      case 'select':
        final current = _values[s.name] as String;
        final options = [...s.options, if (!s.options.any((o) => o.value == current)) SettingOption(current, current)];
        return Padding(
          padding: const EdgeInsets.only(bottom: 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              FieldLabel(s.title),
              DropdownButtonFormField<String>(
                initialValue: current,
                isExpanded: true,
                items: [for (final o in options) DropdownMenuItem(value: o.value, child: Text(o.label))],
                onChanged: (v) => setState(() => _values[s.name] = v ?? current),
              ),
            ],
          ),
        );
      case 'multi-select':
        final selected = (_values[s.name] as List<String>).toSet();
        return Padding(
          padding: const EdgeInsets.only(bottom: 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              FieldLabel(s.title),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final o in s.options)
                    FilterChip(
                      label: Text(o.label),
                      selected: selected.contains(o.value),
                      onSelected: (on) => setState(() {
                        _values[s.name] = [
                          for (final x in s.options)
                            if (x.value == o.value ? on : selected.contains(x.value)) x.value,
                        ];
                      }),
                    ),
                ],
              ),
            ],
          ),
        );
      default:
        final password = s.type == 'password';
        return LabeledField(
          label: s.title,
          controller: _text[s.name]!,
          obscure: password,
          helper: password && d.passwordSet(s.name) ? l.indexerPasswordSet : null,
        );
    }
  }
}
