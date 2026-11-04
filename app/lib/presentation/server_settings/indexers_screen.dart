import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../../core/utils/format.dart';
import '../../data/api/indexers_api.dart';
import '../../data/models/server_settings.dart';
import '../../l10n/l10n.dart';
import '../common/widgets/common.dart';
import 'form.dart';

enum _Filter { all, enabled, public, private }

/// Server trackers: enabling, opening a tracker's settings, updating definitions.
class IndexersScreen extends StatefulWidget {
  const IndexersScreen({super.key});

  @override
  State<IndexersScreen> createState() => _IndexersScreenState();
}

class _IndexersScreenState extends State<IndexersScreen> {
  final _api = getIt<IndexersApi>();
  final _query = TextEditingController();
  List<Indexer>? _items;
  DefinitionsStatus? _defs;
  Object? _error;
  _Filter _filter = _Filter.all;
  bool _updating = false;
  final _toggling = <String>{};

  @override
  void initState() {
    super.initState();
    _query.addListener(() => setState(() {}));
    _load();
  }

  @override
  void dispose() {
    _query.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      final (items, defs) = await (_api.list(), _api.definitions()).wait;
      if (mounted) {
        setState(() {
          _items = items;
          _defs = defs;
        });
      }
    } on ParallelWaitError<(List<Indexer>?, DefinitionsStatus?), (AsyncError?, AsyncError?)> catch (e) {
      if (mounted) setState(() => _error = e.errors.$1?.error ?? e.errors.$2?.error);
    }
  }

  Future<void> _updateDefinitions() async {
    final l = context.l10n;
    setState(() => _updating = true);
    try {
      final d = await _api.updateDefinitions();
      if (mounted) showMessage(context, l.indexersDefinitionsUpdated(d.count));
      await _load();
    } catch (e) {
      if (mounted) showMessage(context, errorText(e));
    } finally {
      if (mounted) setState(() => _updating = false);
    }
  }

  Future<void> _toggle(Indexer ix, bool on) async {
    // enabling a tracker without a login is pointless — open its settings first
    if (on && !ix.configured) return _open(ix);
    setState(() => _toggling.add(ix.id));
    try {
      final d = await _api.save(ix.id, enabled: on);
      _replace(d.indexer);
    } catch (e) {
      if (mounted) showMessage(context, errorText(e));
    } finally {
      if (mounted) setState(() => _toggling.remove(ix.id));
    }
  }

  void _replace(Indexer ix) {
    if (!mounted) return;
    setState(() => _items = [for (final i in _items ?? const <Indexer>[]) i.id == ix.id ? ix : i]);
  }

  Future<void> _open(Indexer ix) async {
    await context.push('/server/indexers/${Uri.encodeComponent(ix.id)}');
    if (mounted) await _load();
  }

  List<Indexer> _visible() {
    final q = _query.text.trim().toLowerCase();
    return [
      for (final i in _items ?? const <Indexer>[])
        if ((q.isEmpty || i.title.toLowerCase().contains(q) || i.id.contains(q)) &&
            switch (_filter) {
              _Filter.all => true,
              _Filter.enabled => i.enabled,
              _Filter.public => i.isPublic,
              _Filter.private => !i.isPublic,
            })
          i,
    ]..sort((a, b) {
      if (a.enabled != b.enabled) return a.enabled ? -1 : 1;
      return a.title.toLowerCase().compareTo(b.title.toLowerCase());
    });
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final items = _items;
    final pad = pagePadding(context);
    return Scaffold(
      appBar: AppBar(title: Text(l.settingsIndexers)),
      body: PageBackground(
        child: Builder(
          builder: (context) {
            if (_error != null) return MessageView.error(_error!, onRetry: _load);
            if (items == null) return const LoadingIndicator();
            final visible = _visible();
            return ListView.builder(
              padding: pad.copyWith(top: 8, bottom: 48),
              itemCount: visible.length + 1,
              itemBuilder: (context, i) {
                if (i > 0) {
                  final ix = visible[i - 1];
                  return _IndexerTile(
                    indexer: ix,
                    busy: _toggling.contains(ix.id),
                    onOpen: () => _open(ix),
                    onToggle: (v) => _toggle(ix, v),
                  );
                }
                final enabled = items.where((i) => i.enabled).length;
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _DefinitionsCard(status: _defs, updating: _updating, onUpdate: _updateDefinitions),
                    const SizedBox(height: 16),
                    TextField(
                      controller: _query,
                      autofocus: true,
                      decoration: InputDecoration(prefixIcon: const Icon(Icons.search), hintText: l.indexersSearch),
                    ),
                    const SizedBox(height: 12),
                    Wrap(
                      spacing: 12,
                      runSpacing: 8,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: [
                        Segmented<_Filter>(
                          options: [
                            (_Filter.all, l.indexersAll),
                            (_Filter.enabled, l.indexersEnabled),
                            (_Filter.public, l.indexersPublic),
                            (_Filter.private, l.indexersPrivate),
                          ],
                          value: _filter,
                          onChanged: (f) => setState(() => _filter = f),
                        ),
                        Text(l.indexersCounter(enabled, items.length), style: AppText.small),
                      ],
                    ),
                    const SizedBox(height: 8),
                    if (visible.isEmpty)
                      Padding(
                        padding: const EdgeInsets.symmetric(vertical: 32),
                        child: Center(
                          child: Text(
                            items.isEmpty ? l.indexersDefinitionsEmpty : l.indexersEmpty,
                            textAlign: TextAlign.center,
                            style: AppText.body.copyWith(color: AppColors.muted),
                          ),
                        ),
                      ),
                  ],
                );
              },
            );
          },
        ),
      ),
    );
  }
}

class _DefinitionsCard extends StatelessWidget {
  final DefinitionsStatus? status;
  final bool updating;
  final VoidCallback onUpdate;
  const _DefinitionsCard({required this.status, required this.updating, required this.onUpdate});

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final s = status ?? const DefinitionsStatus();
    final busy = updating || s.updating;
    return AppCard(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(l.indexersDefinitions, style: AppText.body.copyWith(fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
          Text(
            busy
                ? l.indexersDefinitionsUpdating
                : s.updatedAt == null
                ? l.indexersDefinitionsNever(s.count)
                : l.indexersDefinitionsInfo(s.count, formatDateTime(s.updatedAt!)),
            style: AppText.small,
          ),
          if (s.lastError.isNotEmpty) ...[
            const SizedBox(height: 8),
            AppBanner(tone: Tone.error, icon: Icons.error_outline, message: l.indexersDefinitionsFailed(s.lastError)),
          ],
          const SizedBox(height: 12),
          BusyButton(label: l.indexersDefinitionsUpdate, icon: Icons.download, busy: busy, onPressed: onUpdate),
        ],
      ),
    );
  }
}

class _IndexerTile extends StatelessWidget {
  final Indexer indexer;
  final bool busy;
  final VoidCallback onOpen;
  final ValueChanged<bool> onToggle;

  const _IndexerTile({required this.indexer, required this.busy, required this.onOpen, required this.onToggle});

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final ix = indexer;
    final type = switch (ix.type) {
      'private' => l.indexersTypePrivate,
      'semi-private' => l.indexersTypeSemi,
      _ => l.indexersTypePublic,
    };
    return Row(
      children: [
        Expanded(
          child: ListTile(
            contentPadding: const EdgeInsets.symmetric(horizontal: 4),
            title: Text(ix.title),
            subtitle: Text(
              [type, if (ix.language.isNotEmpty) ix.language].join(' · '),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            trailing: Wrap(
              spacing: 6,
              children: [
                if (!ix.configured) AppBadge(l.indexersNeedsSetup, tone: Tone.warning),
                if (ix.lastError.isNotEmpty) AppBadge(l.indexersError, tone: Tone.error),
              ],
            ),
            onTap: onOpen,
          ),
        ),
        SizedBox(
          width: 64,
          child: busy ? const Center(child: Spinner(size: 16)) : Switch(value: ix.enabled, onChanged: onToggle),
        ),
      ],
    );
  }
}
