import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:media_kit/media_kit.dart';
import 'package:media_kit_video/media_kit_video.dart';

import '../../core/di/injection.dart';
import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../common/widgets/common.dart';
import '../../core/utils/format.dart';
import '../../data/api/torrent_api.dart';
import '../../data/models/torrent.dart';
import '../../data/storage/history_store.dart';
import '../../data/storage/settings_store.dart';
import '../router/app_router.dart';
import 'external_tracks.dart';
import '../../l10n/l10n.dart';

/// Built-in player (libmpv via media_kit).
///
/// Remote: OK — pause, ←/→ — seek (holding speeds it up), ↑/↓ — button panel
/// (tracks, subtitles, next episode), "Back" — close the panel or exit.
/// Phone: tap — panel, double tap left/right — seek.
class PlayerScreen extends StatefulWidget {
  final PlayerArgs args;
  const PlayerScreen({super.key, required this.args});

  @override
  State<PlayerScreen> createState() => _PlayerScreenState();
}

class _PlayerScreenState extends State<PlayerScreen> {
  final _ts = getIt<TorrentApi>();
  final _history = getIt<HistoryStore>();

  late final Player _player = Player(configuration: const PlayerConfiguration(bufferSize: 64 * 1024 * 1024));

  /// media_kit sets mpv's network-timeout to 5 s, but the first byte of a stream can take longer:
  /// the server first loads a saved torrent back into the engine (up to a minute for metadata).
  /// mpv then gave up on a stream an external player would play.
  late final Future<void> _configured = () async {
    final native = _player.platform;
    if (native is NativePlayer) await native.setProperty('network-timeout', '90');
  }();

  /// Android: MediaCodec renders straight into a Surface — 10-bit, HDR10/HDR10+/Dolby Vision; in the
  /// default mode (vo=gpu, hwdec=auto-safe, copy into a GLES texture) such video may come out black.
  /// media_kit renders text subtitles in Flutter, so they work in both modes.
  late final bool _hwOutput = Platform.isAndroid && getIt<SettingsStore>().videoOutput == VideoOutput.hardware;
  late final VideoController _video = VideoController(
    _player,
    configuration: _hwOutput
        ? const VideoControllerConfiguration(vo: 'mediacodec_embed', hwdec: 'mediacodec')
        : const VideoControllerConfiguration(),
  );

  /// All files of the release — the source of external subtitles and audio tracks.
  late List<TorrentFile> _files = widget.args.files;
  bool _outputHintShown = false;

  final _rootFocus = FocusNode(debugLabel: 'player');
  final _firstButtonFocus = FocusNode(debugLabel: 'player-buttons');
  final _subs = <StreamSubscription<dynamic>>[];

  late TorrentFile _file = widget.args.file;
  bool _overlay = true;
  bool _buttonsFocused = false;
  Timer? _hideTimer;
  Timer? _saveTimer;
  Timer? _statTimer;
  Torrent? _stat;
  int _repeat = 0;
  String? _seekHint;
  Timer? _seekHintTimer;

  TorrentFile? get _next {
    final list = widget.args.playlist;
    final i = list.indexWhere((f) => f.id == _file.id);
    return (i >= 0 && i + 1 < list.length) ? list[i + 1] : null;
  }

  @override
  void initState() {
    super.initState();
    SystemChrome.setEnabledSystemUIMode(SystemUiMode.immersiveSticky);
    _subs.add(
      _player.stream.completed.listen((done) {
        if (done) _onCompleted();
      }),
    );
    _subs.add(_player.stream.playing.listen((_) => _scheduleHide()));
    _subs.add(_player.stream.error.listen(_onError));
    if (_files.isEmpty) _loadFiles();
    _saveTimer = Timer.periodic(const Duration(seconds: 5), (_) => _save());
    _statTimer = Timer.periodic(const Duration(seconds: 2), (_) => _pollStat());
    _open(_file);
  }

  @override
  void dispose() {
    _save();
    for (final s in _subs) {
      s.cancel();
    }
    _hideTimer?.cancel();
    _saveTimer?.cancel();
    _statTimer?.cancel();
    _seekHintTimer?.cancel();
    _player.dispose();
    _rootFocus.dispose();
    _firstButtonFocus.dispose();
    SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
    super.dispose();
  }

  Future<void> _open(TorrentFile f) async {
    _file = f;
    final e = _history.entry(widget.args.hash, f.id);
    // resume where playback stopped, stepping back a couple of seconds
    final start = (e != null && !e.finished && e.positionMs > 10000)
        ? Duration(milliseconds: e.positionMs - 3000)
        : null;
    await _configured;
    await _player.open(Media(_ts.streamUrl(widget.args.hash, f), start: start));
    if (mounted) setState(() {});
  }

  Future<void> _loadFiles() async {
    try {
      final st = await _ts.get(widget.args.hash);
      if (mounted) setState(() => _files = st.files);
    } catch (_) {}
  }

  /// Decoder/output error in MediaCodec mode — suggest the compatible mode (once).
  void _onError(String text) {
    if (!_hwOutput || _outputHintShown || !mounted || text.startsWith('tcp:')) return;
    _outputHintShown = true;
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(context.l10n.playerVideoOutputHint), duration: const Duration(seconds: 10)));
  }

  void _save() {
    final pos = _player.state.position;
    final dur = _player.state.duration;
    if (dur <= Duration.zero || pos <= Duration.zero) return;
    _history.save(
      WatchEntry(
        media: widget.args.media,
        hash: widget.args.hash,
        fileId: _file.id,
        fileName: _file.name,
        torrentTitle: widget.args.torrentTitle,
        positionMs: pos.inMilliseconds,
        durationMs: dur.inMilliseconds,
        updatedAt: DateTime.now(),
      ),
    );
  }

  Future<void> _pollStat() async {
    if (!_overlay && !_player.state.buffering) return;
    try {
      final st = await _ts.get(widget.args.hash);
      if (mounted) setState(() => _stat = st);
    } catch (_) {}
  }

  void _onCompleted() {
    _save();
    final next = _next;
    if (next != null) {
      _open(next);
      _showOverlay();
    } else if (mounted) {
      Navigator.of(context).maybePop();
    }
  }

  void _showOverlay() {
    if (!_overlay) setState(() => _overlay = true);
    _scheduleHide();
  }

  void _scheduleHide() {
    _hideTimer?.cancel();
    _hideTimer = Timer(const Duration(seconds: 4), () {
      if (mounted && _player.state.playing && !_buttonsFocused) setState(() => _overlay = false);
    });
  }

  void _hideOverlay() {
    _rootFocus.requestFocus();
    setState(() {
      _overlay = false;
      _buttonsFocused = false;
    });
  }

  Future<void> _seekBy(Duration delta) async {
    final dur = _player.state.duration;
    var target = _player.state.position + delta;
    if (target < Duration.zero) target = Duration.zero;
    if (dur > Duration.zero && target > dur) target = dur - const Duration(seconds: 1);
    await _player.seek(target);
    _seekHintTimer?.cancel();
    setState(() => _seekHint = context.l10n.seconds('${delta.isNegative ? '−' : '+'}${delta.inSeconds.abs()}'));
    _seekHintTimer = Timer(const Duration(milliseconds: 800), () {
      if (mounted) setState(() => _seekHint = null);
    });
    _showOverlay();
  }

  /// The seek step grows while the key is held: 10 s → 30 s → 60 s.
  Duration get _step => Duration(seconds: _repeat < 5 ? 10 : (_repeat < 15 ? 30 : 60));

  KeyEventResult _onKey(FocusNode node, KeyEvent event) {
    // Focus is on the panel buttons — arrows and OK work as normal navigation.
    if (!_rootFocus.hasPrimaryFocus) return KeyEventResult.ignored;
    if (event is KeyUpEvent) {
      _repeat = 0;
      return KeyEventResult.ignored;
    }
    if (event is KeyRepeatEvent) _repeat++;
    final key = event.logicalKey;

    if (key == LogicalKeyboardKey.arrowLeft || key == LogicalKeyboardKey.mediaRewind) {
      _seekBy(-_step);
      return KeyEventResult.handled;
    }
    if (key == LogicalKeyboardKey.arrowRight || key == LogicalKeyboardKey.mediaFastForward) {
      _seekBy(_step);
      return KeyEventResult.handled;
    }
    if (event is KeyRepeatEvent) return KeyEventResult.handled;

    if (key == LogicalKeyboardKey.select ||
        key == LogicalKeyboardKey.enter ||
        key == LogicalKeyboardKey.space ||
        key == LogicalKeyboardKey.mediaPlayPause) {
      _player.playOrPause();
      _showOverlay();
      return KeyEventResult.handled;
    }
    if (key == LogicalKeyboardKey.mediaPlay) {
      _player.play();
      return KeyEventResult.handled;
    }
    if (key == LogicalKeyboardKey.mediaPause) {
      _player.pause();
      _showOverlay();
      return KeyEventResult.handled;
    }
    if (key == LogicalKeyboardKey.mediaTrackNext && _next != null) {
      _save();
      _open(_next!);
      return KeyEventResult.handled;
    }
    if (key == LogicalKeyboardKey.arrowDown || key == LogicalKeyboardKey.arrowUp) {
      setState(() => _overlay = true);
      _hideTimer?.cancel();
      _firstButtonFocus.requestFocus();
      return KeyEventResult.handled;
    }
    return KeyEventResult.ignored;
  }

  /// Is the external file already loaded into mpv (audio-add/sub-add use its name as the title)?
  static bool _loaded(List<dynamic> tracks, String name) => tracks.any((t) => t.uri != true && t.title == name);

  /// Same track: the external one chosen by URI and its entry in mpv's list match by name.
  static bool _same(dynamic a, dynamic b) =>
      a == b || ((a.uri == true || b.uri == true) && a.title != null && a.title == b.title);

  Future<void> _chooseAudio() async {
    final own = _player.state.tracks.audio.where((t) => t.id != 'auto' && t.id != 'no').toList();
    // external tracks (mka, ac3…) are loaded only when selected — nothing extra is downloaded
    final external = [
      for (final e in externalAudio(_file, _files))
        if (!_loaded(own, e.file.name))
          AudioTrack.uri(_ts.streamUrl(widget.args.hash, e.file), title: e.file.name, language: e.language),
    ];
    final current = _player.state.track.audio;
    final picked = await _pickTrack<AudioTrack>(
      context.l10n.playerAudioTrack,
      [...own, ...external],
      current,
      _trackLabel,
    );
    if (picked != null) await _player.setAudioTrack(picked);
  }

  Future<void> _chooseSubtitles() async {
    final own = _player.state.tracks.subtitle.where((t) => t.id != 'auto' && t.id != 'no').toList();
    final external = [
      for (final e in externalSubtitles(_file, _files))
        if (!_loaded(own, e.file.name))
          SubtitleTrack.uri(_ts.streamUrl(widget.args.hash, e.file), title: e.file.name, language: e.language),
    ];
    final current = _player.state.track.subtitle;
    final picked = await _pickTrack<SubtitleTrack>(
      context.l10n.playerSubtitles,
      [SubtitleTrack.no(), ...own, ...external],
      current,
      (t) => t.id == 'no' ? context.l10n.playerSubtitlesOff : _trackLabel(t),
    );
    if (picked != null) await _player.setSubtitleTrack(picked);
  }

  String _trackLabel(dynamic t) {
    final parts = [
      t.title as String?,
      (t.language as String?)?.toUpperCase(),
    ].whereType<String>().where((s) => s.isNotEmpty);
    return parts.isEmpty ? context.l10n.playerTrack('${t.id}') : parts.join(' · ');
  }

  /// Short label of the selected track for the panel button; null — mpv decides.
  static String? _shortLabel(dynamic t) {
    if (t.id == 'auto') return null;
    final lang = (t.language as String?)?.toUpperCase() ?? '';
    if (lang.isNotEmpty) return lang;
    final title = (t.title as String?) ?? '';
    if (title.isNotEmpty) return title.length > 14 ? '${title.substring(0, 13)}…' : title;
    return '#${t.id}';
  }

  /// Track picker; with a remote, focus starts on the current track.
  Future<T?> _pickTrack<T>(String title, List<T> tracks, T current, String Function(T) label) {
    final currentIndex = tracks.indexWhere((t) => _same(t, current));
    return showDialog<T>(
      context: context,
      builder: (ctx) => SimpleDialog(
        title: Text(title),
        children: [
          if (tracks.isEmpty) Padding(padding: const EdgeInsets.all(24), child: Text(ctx.l10n.playerNoTracks)),
          for (final (i, t) in tracks.indexed)
            ListTile(
              autofocus: i == (currentIndex < 0 ? 0 : currentIndex),
              contentPadding: const EdgeInsets.symmetric(horizontal: 24),
              onTap: () => Navigator.pop(ctx, t),
              leading: Icon(
                i == currentIndex ? Icons.radio_button_checked : Icons.radio_button_off,
                color: i == currentIndex ? AppColors.accent : AppColors.muted,
                size: 20,
              ),
              title: Text(label(t)),
              trailing: (t as dynamic).uri == true ? const Icon(Icons.attach_file, size: 18) : null,
            ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final title = widget.args.media?.title ?? widget.args.torrentTitle;
    return PopScope(
      // "Back" first moves focus off the panel buttons, and only then closes the player.
      canPop: !_buttonsFocused,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _hideOverlay();
      },
      child: Scaffold(
        backgroundColor: Colors.black,
        body: Focus(
          focusNode: _rootFocus,
          autofocus: true,
          onKeyEvent: _onKey,
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: () => _overlay ? setState(() => _overlay = false) : _showOverlay(),
            onDoubleTapDown: (d) {
              final left = d.localPosition.dx < MediaQuery.sizeOf(context).width / 2;
              _seekBy(Duration(seconds: left ? -10 : 10));
            },
            onDoubleTap: () {},
            child: Stack(
              fit: StackFit.expand,
              children: [
                Video(controller: _video, controls: NoVideoControls),
                _BufferingIndicator(player: _player, stat: _stat),
                if (_seekHint != null)
                  Center(
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
                      decoration: BoxDecoration(
                        color: AppColors.bg.withValues(alpha: 0.85),
                        border: Border.all(color: AppColors.border),
                        borderRadius: BorderRadius.circular(AppRadius.control),
                      ),
                      child: Text(_seekHint!, style: AppText.statValue),
                    ),
                  ),
                AnimatedOpacity(
                  opacity: _overlay ? 1 : 0,
                  duration: const Duration(milliseconds: 200),
                  child: IgnorePointer(
                    ignoring: !_overlay,
                    child: _Overlay(
                      player: _player,
                      title: title,
                      subtitle: _file.name,
                      stat: _stat,
                      firstButtonFocus: _firstButtonFocus,
                      hasNext: _next != null,
                      onBack: () => Navigator.of(context).maybePop(),
                      onSeekBy: _seekBy,
                      onAudio: _chooseAudio,
                      onSubtitles: _chooseSubtitles,
                      onNext: () {
                        _save();
                        _open(_next!);
                      },
                      onButtonsFocus: (f) {
                        setState(() => _buttonsFocused = f);
                        if (!f) _scheduleHide();
                      },
                      onLeaveButtons: _hideOverlay,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _BufferingIndicator extends StatelessWidget {
  final Player player;
  final Torrent? stat;
  const _BufferingIndicator({required this.player, required this.stat});

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<bool>(
      stream: player.stream.buffering,
      initialData: player.state.buffering,
      builder: (_, snap) {
        if (snap.data != true) return const SizedBox.shrink();
        final s = stat;
        return Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const IconTheme(
                data: IconThemeData(color: AppColors.accent),
                child: Spinner(size: 48),
              ),
              if (s != null) ...[
                const SizedBox(height: 16),
                Text(
                  context.l10n.playerPeers(formatSpeed(s.downloadSpeed), '${s.activePeers}/${s.totalPeers}'),
                  style: AppText.body.copyWith(color: AppColors.muted),
                ),
              ],
            ],
          ),
        );
      },
    );
  }
}

class _Overlay extends StatelessWidget {
  final Player player;
  final String title;
  final String subtitle;
  final Torrent? stat;
  final FocusNode firstButtonFocus;
  final bool hasNext;
  final VoidCallback onBack;
  final void Function(Duration) onSeekBy;
  final VoidCallback onAudio;
  final VoidCallback onSubtitles;
  final VoidCallback onNext;
  final void Function(bool) onButtonsFocus;
  final VoidCallback onLeaveButtons;

  const _Overlay({
    required this.player,
    required this.title,
    required this.subtitle,
    required this.stat,
    required this.firstButtonFocus,
    required this.hasNext,
    required this.onBack,
    required this.onSeekBy,
    required this.onAudio,
    required this.onSubtitles,
    required this.onNext,
    required this.onButtonsFocus,
    required this.onLeaveButtons,
  });

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [Colors.black87, Colors.transparent, Colors.transparent, Colors.black87],
          stops: [0, 0.25, 0.6, 1],
        ),
      ),
      child: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                children: [
                  ExcludeFocus(
                    child: IconButton(
                      onPressed: onBack,
                      icon: const Icon(Icons.arrow_back, color: AppColors.text),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(title, style: AppText.pageTitle, maxLines: 1, overflow: TextOverflow.ellipsis),
                        Text(subtitle, style: AppText.small, maxLines: 1, overflow: TextOverflow.ellipsis),
                      ],
                    ),
                  ),
                  if (stat != null)
                    Text(
                      context.l10n.playerPeers(formatSpeed(stat!.downloadSpeed), '${stat!.activePeers}'),
                      style: AppText.small,
                    ),
                ],
              ),
              const Spacer(),
              _Progress(player: player),
              const SizedBox(height: 8),
              Focus(
                skipTraversal: true,
                canRequestFocus: false,
                onFocusChange: onButtonsFocus,
                onKeyEvent: (_, e) {
                  if (e is KeyDownEvent && e.logicalKey == LogicalKeyboardKey.arrowUp) {
                    onLeaveButtons();
                    return KeyEventResult.handled;
                  }
                  return KeyEventResult.ignored;
                },
                child: FocusTraversalGroup(
                  child: Row(
                    children: [
                      StreamBuilder<bool>(
                        stream: player.stream.playing,
                        initialData: player.state.playing,
                        builder: (_, s) => _Btn(
                          focusNode: firstButtonFocus,
                          icon: s.data == true ? Icons.pause : Icons.play_arrow,
                          label: s.data == true ? context.l10n.playerPause : context.l10n.playerPlay,
                          onTap: player.playOrPause,
                        ),
                      ),
                      _Btn(
                        icon: Icons.replay_10,
                        label: context.l10n.seconds('−10'),
                        onTap: () => onSeekBy(const Duration(seconds: -10)),
                      ),
                      _Btn(
                        icon: Icons.forward_30,
                        label: context.l10n.seconds('+30'),
                        onTap: () => onSeekBy(const Duration(seconds: 30)),
                      ),
                      const Spacer(),
                      StreamBuilder<Track>(
                        stream: player.stream.track,
                        initialData: player.state.track,
                        builder: (_, s) {
                          final audio = _PlayerScreenState._shortLabel(s.data!.audio);
                          final sub = s.data!.subtitle;
                          final subLabel = sub.id == 'no'
                              ? context.l10n.playerSubtitlesOff
                              : _PlayerScreenState._shortLabel(sub);
                          return Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              _Btn(
                                icon: Icons.audiotrack,
                                label: [context.l10n.playerAudio, ?audio].join(' · '),
                                onTap: onAudio,
                              ),
                              _Btn(
                                icon: Icons.subtitles,
                                label: [context.l10n.playerSubtitles, ?subLabel].join(' · '),
                                onTap: onSubtitles,
                              ),
                            ],
                          );
                        },
                      ),
                      if (hasNext) _Btn(icon: Icons.skip_next, label: context.l10n.playerNext, onTap: onNext),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Btn extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;
  final FocusNode? focusNode;

  const _Btn({required this.icon, required this.label, required this.onTap, this.focusNode});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(right: 8),
      child: TextButton.icon(
        focusNode: focusNode,
        onPressed: onTap,
        icon: Icon(icon),
        label: Text(label),
        // Over the video: light text; when focused — like a cyan button with a bright border.
        style: AppButtons.of(ButtonVariant.neutral).copyWith(
          foregroundColor: WidgetStateProperty.resolveWith(
            (s) => s.contains(WidgetState.focused) ? AppColors.accent : AppColors.text,
          ),
          iconColor: WidgetStateProperty.resolveWith(
            (s) => s.contains(WidgetState.focused) ? AppColors.accent : AppColors.text,
          ),
          backgroundColor: WidgetStateProperty.resolveWith(
            (s) => s.contains(WidgetState.focused)
                ? AppColors.accent.withValues(alpha: AppAlpha.ghostHover)
                : AppColors.bg.withValues(alpha: 0.6),
          ),
        ),
      ),
    );
  }
}

class _Progress extends StatefulWidget {
  final Player player;
  const _Progress({required this.player});

  @override
  State<_Progress> createState() => _ProgressState();
}

class _ProgressState extends State<_Progress> {
  double? _drag;

  @override
  Widget build(BuildContext context) {
    final p = widget.player;
    return StreamBuilder<Duration>(
      stream: p.stream.position,
      initialData: p.state.position,
      builder: (_, posSnap) {
        final dur = p.state.duration;
        final pos = posSnap.data ?? Duration.zero;
        final max = dur.inMilliseconds.toDouble();
        final value = _drag ?? pos.inMilliseconds.clamp(0, max <= 0 ? 0 : max).toDouble();
        final buffer = p.state.buffer.inMilliseconds.toDouble();
        return Row(
          children: [
            Text(formatDuration(_drag != null ? Duration(milliseconds: _drag!.round()) : pos), style: AppText.small),
            Expanded(
              child: Stack(
                alignment: Alignment.center,
                children: [
                  if (max > 0)
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 24),
                      child: LinearProgressIndicator(
                        value: (buffer / max).clamp(0, 1),
                        minHeight: 3,
                        color: AppColors.muted.withValues(alpha: 0.5),
                        backgroundColor: AppColors.border,
                      ),
                    ),
                  // With a remote, seeking uses the arrows; the slider is for touch only.
                  ExcludeFocus(
                    child: Slider(
                      value: max <= 0 ? 0 : value,
                      max: max <= 0 ? 1 : max,
                      activeColor: AppColors.accent,
                      inactiveColor: Colors.transparent,
                      onChanged: max <= 0 ? null : (v) => setState(() => _drag = v),
                      onChangeEnd: (v) {
                        p.seek(Duration(milliseconds: v.round()));
                        setState(() => _drag = null);
                      },
                    ),
                  ),
                ],
              ),
            ),
            Text(formatDuration(dur), style: AppText.small),
          ],
        );
      },
    );
  }
}
