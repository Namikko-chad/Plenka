import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';

import '../../core/config/app_config.dart';
import '../../core/di/injection.dart';
import '../../data/storage/settings_store.dart';
import '../../l10n/l10n.dart';
import '../common/widgets/common.dart';
import '../connect/connection_cubit.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  final _settings = getIt<SettingsStore>();

  @override
  void initState() {
    super.initState();
    context.read<ConnectionCubit>().refresh();
  }

  /// "Change server": the address is remembered and tried first on later launches.
  Future<void> _changeServer(String current) async {
    final controller = TextEditingController(text: current);
    final cubit = context.read<ConnectionCubit>();
    final address = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(ctx.l10n.settingsChangeServer),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            FieldLabel(ctx.l10n.settingsServerAddress),
            TextField(
              controller: controller,
              autofocus: true,
              keyboardType: TextInputType.url,
              decoration: InputDecoration(hintText: ctx.l10n.connectAddressHint(AppConfig.serverPort)),
              onSubmitted: (v) => Navigator.pop(ctx, v),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: Text(ctx.l10n.cancel)),
          FilledButton(onPressed: () => Navigator.pop(ctx, controller.text), child: Text(ctx.l10n.settingsConnect)),
        ],
      ),
    );
    controller.dispose();
    if (address == null || address.trim().isEmpty) return;
    final ok = await cubit.connectTo(address);
    if (!ok && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(context.l10n.connectNoResponse)));
    }
  }

  /// Server settings screen; the server statuses are reloaded on return.
  Future<void> _open(String path) async {
    final cubit = context.read<ConnectionCubit>();
    await context.push(path);
    await cubit.refresh();
  }

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<ConnectionCubit, ServerConnectionState>(
      builder: (context, state) {
        final l = context.l10n;
        final info = state.info;
        final cubit = context.read<ConnectionCubit>();
        return ListView(
          padding: pagePadding(context).copyWith(top: 32, bottom: 60),
          children: [
            PageTitle(l.settingsTitle),
            SectionTitle(l.settingsServer, inset: false),
            AppCard(
              padding: const EdgeInsets.all(8),
              child: Column(
                children: [
                  ListTile(
                    leading: const Icon(Icons.router_outlined, size: 20),
                    title: Text(info?.baseUrl ?? l.settingsNotConnected),
                    subtitle: Text(
                      [
                        if (info != null && info.version.isNotEmpty) 'plenka-server ${info.version}',
                        _settings.serverManual ? l.settingsServerManual : l.settingsServerAuto,
                      ].join(' · '),
                    ),
                  ),
                  ListTile(
                    autofocus: true,
                    leading: const Icon(Icons.edit_outlined, size: 20),
                    title: Text(l.settingsChangeServer),
                    subtitle: Text(l.settingsChangeServerHint),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () => _changeServer(info?.baseUrl ?? _settings.serverUrl ?? ''),
                  ),
                  ListTile(
                    leading: const Icon(Icons.wifi_find, size: 20),
                    title: Text(l.settingsFindAuto),
                    onTap: cubit.discoverAutomatically,
                  ),
                  if (info != null) ...[
                    const Divider(height: 1),
                    _Status(
                      l.settingsTorrents,
                      info.torrents,
                      info.torrents ? l.settingsTorrentsOk : l.settingsTorrentsOff,
                    ),
                    _Status(
                      l.settingsIndexers,
                      info.indexersEnabled > 0,
                      l.settingsIndexersCount(info.indexersEnabled, info.indexersTotal),
                    ),
                    _Status('TMDB', info.tmdb, info.tmdb ? l.settingsTmdbOk : l.settingsTmdbOff),
                  ],
                ],
              ),
            ),
            if (info != null) ...[
              SectionTitle(l.settingsServerSettings, inset: false),
              AppCard(
                padding: const EdgeInsets.all(8),
                child: Column(
                  children: [
                    _NavTile(
                      icon: Icons.travel_explore,
                      title: l.settingsIndexers,
                      subtitle: l.settingsIndexersHint,
                      onTap: () => _open('/server/indexers'),
                    ),
                    _NavTile(
                      icon: Icons.movie_filter_outlined,
                      title: l.settingsCatalog,
                      subtitle: l.settingsCatalogHint,
                      onTap: () => _open('/server/catalog'),
                    ),
                    _NavTile(
                      icon: Icons.tune,
                      title: l.settingsEngine,
                      subtitle: l.settingsEngineHint,
                      onTap: () => _open('/server/engine'),
                    ),
                  ],
                ),
              ),
            ],
            SectionTitle(l.settingsLanguage, inset: false),
            AppCard(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: RadioGroup<AppLanguage>(
                groupValue: _settings.language.value,
                onChanged: (v) {
                  if (v != null) _settings.setLanguage(v);
                },
                child: Column(
                  children: [
                    for (final v in AppLanguage.values)
                      RadioListTile<AppLanguage>(value: v, title: Text(_languageName(l, v))),
                  ],
                ),
              ),
            ),
            SectionTitle(l.settingsPlayer, inset: false),
            AppCard(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: RadioGroup<PlayerMode>(
                groupValue: _settings.playerMode,
                onChanged: (m) async {
                  if (m == null) return;
                  await _settings.setPlayerMode(m);
                  setState(() {});
                },
                child: Column(
                  children: [
                    for (final m in PlayerMode.values)
                      RadioListTile<PlayerMode>(
                        value: m,
                        title: Text(_playerModeName(l, m)),
                        enabled: m == PlayerMode.internal || Platform.isAndroid,
                      ),
                  ],
                ),
              ),
            ),
            // On Linux media_kit has its own output (libmpv into a texture) — nothing to choose.
            if (Platform.isAndroid) ...[
              SectionTitle(l.settingsVideoOutput, inset: false),
              AppCard(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: RadioGroup<VideoOutput>(
                  groupValue: _settings.videoOutput,
                  onChanged: (v) async {
                    if (v == null) return;
                    await _settings.setVideoOutput(v);
                    setState(() {});
                  },
                  child: Column(
                    children: [
                      RadioListTile<VideoOutput>(
                        value: VideoOutput.hardware,
                        title: Text(l.videoOutputHardware),
                        subtitle: Text(l.videoOutputHardwareHint),
                      ),
                      RadioListTile<VideoOutput>(
                        value: VideoOutput.compatible,
                        title: Text(l.videoOutputCompatible),
                        subtitle: Text(l.videoOutputCompatibleHint),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ],
        );
      },
    );
  }
}

/// Language name in that language itself, so someone who can't read the current one recognizes it.
String _languageName(AppLocalizations l, AppLanguage v) => switch (v) {
  AppLanguage.system => l.languageSystem,
  AppLanguage.en => 'English',
  AppLanguage.ru => 'Русский',
};

String _playerModeName(AppLocalizations l, PlayerMode m) => switch (m) {
  PlayerMode.internal => l.playerModeInternal,
  PlayerMode.external => l.playerModeExternal,
  PlayerMode.ask => l.playerModeAsk,
};

class _NavTile extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;
  const _NavTile({required this.icon, required this.title, required this.subtitle, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Icon(icon, size: 20),
      title: Text(title),
      subtitle: Text(subtitle),
      trailing: const Icon(Icons.chevron_right),
      onTap: onTap,
    );
  }
}

class _Status extends StatelessWidget {
  final String name;
  final bool ok;
  final String hint;
  const _Status(this.name, this.ok, this.hint);

  @override
  Widget build(BuildContext context) {
    return ListTile(
      dense: true,
      leading: SizedBox(
        width: 20,
        child: Center(child: StatusDot(ok: ok)),
      ),
      title: Text(name),
      subtitle: Text(hint),
      trailing: AppBadge(ok ? context.l10n.statusOk : context.l10n.statusError, tone: ok ? Tone.ok : Tone.error),
    );
  }
}
