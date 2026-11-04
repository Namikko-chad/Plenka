import 'dart:math' as math;

import 'package:cached_network_image/cached_network_image.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';

import '../../../core/theme/app_theme.dart';
import '../../../core/theme/colors.dart';
import '../../../l10n/l10n.dart';

/// Large screen (TV, tablet in landscape): side menu and large posters.
bool isWide(BuildContext context) {
  final size = MediaQuery.sizeOf(context);
  return size.width >= 840 || (size.width > size.height && size.shortestSide >= 500);
}

double posterWidth(BuildContext context) => isWide(context) ? 150 : 112;

/// Page padding: 40 on a large screen, 16 on a phone (DESIGN_MANIFEST.md § 2.5–2.6).
EdgeInsets pagePadding(BuildContext context) =>
    isWide(context) ? const EdgeInsets.symmetric(horizontal: 40) : const EdgeInsets.symmetric(horizontal: 16);

String errorText(Object e) {
  if (e is DioException) {
    final code = e.response?.statusCode;
    final tmdb = e.requestOptions.path.contains('/api/tmdb/');
    if (tmdb && code == 503) return tr.errorTmdbNotConfigured;
    if (tmdb && code == 401) return tr.errorTmdbKeyRejected;
    if (code != null) {
      // native /api/* respond with {"error": "…"}
      final body = e.response?.data;
      final msg = body is Map ? body['error'] : null;
      return msg is String && msg.isNotEmpty ? tr.errorServerMessage(code, msg) : tr.errorServerCode('$code');
    }
    if (e.type == DioExceptionType.receiveTimeout) return tr.errorServerSlow;
    return tr.errorServerUnreachable;
  }
  final s = e.toString();
  return s.startsWith('Exception: ') ? s.substring(11) : s;
}

// ─── Page background ───────────────────────────────────────────────────────

/// Dark background with a fixed 40 px cyan grid (§ 2.2). Wraps the body of every screen.
class PageBackground extends StatelessWidget {
  final Widget child;
  const PageBackground({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return CustomPaint(painter: const _GridPainter(), isComplex: true, child: child);
  }
}

class _GridPainter extends CustomPainter {
  const _GridPainter();
  static const double _cell = 40;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.drawRect(Offset.zero & size, Paint()..color = AppColors.bg);
    final line = Paint()
      ..color = AppColors.accent.withValues(alpha: AppAlpha.grid)
      ..strokeWidth = 1;
    for (double x = 0; x < size.width; x += _cell) {
      canvas.drawLine(Offset(x + 0.5, 0), Offset(x + 0.5, size.height), line);
    }
    for (double y = 0; y < size.height; y += _cell) {
      canvas.drawLine(Offset(0, y + 0.5), Offset(size.width, y + 0.5), line);
    }
  }

  @override
  bool shouldRepaint(covariant _GridPainter oldDelegate) => false;
}

// ─── Headings ──────────────────────────────────────────────────────────────

/// Page title: 22 px Syne 800 (§ 3.3).
class PageTitle extends StatelessWidget {
  final String text;
  const PageTitle(this.text, {super.key});

  @override
  Widget build(BuildContext context) => Text(text, style: AppText.pageTitle);
}

/// Section heading: uppercase, letter-spaced, a line to the edge (§ 18.1).
/// [inset] — add page padding (for full-width rows).
class SectionTitle extends StatelessWidget {
  final String text;
  final bool inset;
  const SectionTitle(this.text, {super.key, this.inset = true});

  @override
  Widget build(BuildContext context) {
    final pad = inset ? pagePadding(context) : EdgeInsets.zero;
    return Padding(
      padding: pad.copyWith(top: 28, bottom: 12),
      child: Row(
        children: [
          Text(text.toUpperCase(), style: AppText.section),
          const SizedBox(width: 12),
          const Expanded(
            child: SizedBox(height: 1, child: ColoredBox(color: AppColors.border)),
          ),
        ],
      ),
    );
  }
}

/// Form field label: 12 px, uppercase, muted (§ 10.3).
class FieldLabel extends StatelessWidget {
  final String text;
  const FieldLabel(this.text, {super.key});

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 6),
    child: Text(text.toUpperCase(), style: AppText.label),
  );
}

// ─── Loading and progress ──────────────────────────────────────────────────

/// The only spinner (§ 17.1): a 3/4 ring, 0.8 s per turn.
/// The color comes from IconTheme — like currentColor on the web.
class Spinner extends StatefulWidget {
  final double size;
  const Spinner({super.key, this.size = 16});

  @override
  State<Spinner> createState() => _SpinnerState();
}

class _SpinnerState extends State<Spinner> with SingleTickerProviderStateMixin {
  late final _c = AnimationController(vsync: this, duration: const Duration(milliseconds: 800))..repeat();

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final color = IconTheme.of(context).color ?? AppColors.accent;
    return RotationTransition(
      turns: _c,
      child: CustomPaint(size: Size.square(widget.size), painter: _ArcPainter(color)),
    );
  }
}

class _ArcPainter extends CustomPainter {
  final Color color;
  const _ArcPainter(this.color);

  @override
  void paint(Canvas canvas, Size size) {
    final p = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2;
    canvas.drawArc((Offset.zero & size).deflate(1), 0, math.pi * 1.5, false, p);
  }

  @override
  bool shouldRepaint(covariant _ArcPainter old) => old.color != color;
}

class LoadingIndicator extends StatelessWidget {
  final String? message;
  const LoadingIndicator({super.key, this.message});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(40),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const IconTheme(
              data: IconThemeData(color: AppColors.accent),
              child: Spinner(size: 48),
            ),
            if (message != null) ...[
              const SizedBox(height: 20),
              Text(
                message!,
                style: AppText.body.copyWith(color: AppColors.muted),
                textAlign: TextAlign.center,
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// 3 px progress bar on a border-colored track (§ 17.3).
class ProgressLine extends StatelessWidget {
  final double value;
  final Color color;
  const ProgressLine({super.key, required this.value, this.color = AppColors.accent});

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(2),
      child: LinearProgressIndicator(
        value: value.clamp(0, 1),
        minHeight: 3,
        color: color,
        backgroundColor: AppColors.border,
      ),
    );
  }
}

// ─── Empty states and banners ──────────────────────────────────────────────

/// Empty state (§ 19.1): a 32 px icon, one line, at most one button.
/// An error is shown as a red banner (§ 19.2) with a "Retry" button.
class MessageView extends StatelessWidget {
  final IconData icon;
  final String message;
  final String? actionLabel;
  final VoidCallback? onAction;
  final bool isError;

  const MessageView({
    super.key,
    required this.icon,
    required this.message,
    this.actionLabel,
    this.onAction,
    this.isError = false,
  });

  factory MessageView.error(Object error, {VoidCallback? onRetry}) => MessageView(
    icon: Icons.error_outline,
    message: errorText(error),
    actionLabel: onRetry == null ? null : tr.retry,
    onAction: onRetry,
    isError: true,
  );

  @override
  Widget build(BuildContext context) {
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(40),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (isError)
                AppBanner(tone: Tone.error, icon: icon, message: message)
              else ...[
                Icon(icon, size: 32, color: AppColors.muted),
                const SizedBox(height: 12),
                Text(
                  message,
                  style: AppText.body.copyWith(color: AppColors.muted),
                  textAlign: TextAlign.center,
                ),
              ],
              if (onAction != null) ...[
                const SizedBox(height: 20),
                OutlinedButton(autofocus: true, onPressed: onAction, child: Text(actionLabel ?? 'OK')),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// Status color tones (§ 1.2).
enum Tone {
  ok(AppColors.ok),
  error(AppColors.error),
  warning(AppColors.warning),
  accent(AppColors.accent),
  neutral(AppColors.paused);

  final Color color;
  const Tone(this.color);
}

/// Page-level banner (§ 19.2).
class AppBanner extends StatelessWidget {
  final Tone tone;
  final IconData? icon;
  final String message;
  const AppBanner({super.key, required this.tone, required this.message, this.icon});

  @override
  Widget build(BuildContext context) {
    final c = tone.color;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: c.withValues(alpha: AppAlpha.tint),
        border: Border.all(color: c.withValues(alpha: AppAlpha.line)),
        borderRadius: BorderRadius.circular(AppRadius.control),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (icon != null) ...[
            Padding(
              padding: const EdgeInsets.only(top: 1),
              child: Icon(icon, size: 16, color: c),
            ),
            const SizedBox(width: 8),
          ],
          Flexible(
            child: Text(
              message,
              style: TextStyle(fontFamily: AppText.mono, fontSize: 12, color: c, height: 1.4),
            ),
          ),
        ],
      ),
    );
  }
}

// ─── Badges and status dots ────────────────────────────────────────────────

/// Badge (§ 12.1): one uppercase word, 15 % background, 30 % border.
/// [onImage] — add a dark backing so the badge reads on top of a poster.
class AppBadge extends StatelessWidget {
  final String text;
  final Tone tone;
  final bool onImage;
  const AppBadge(this.text, {super.key, this.tone = Tone.neutral, this.onImage = false});

  @override
  Widget build(BuildContext context) {
    final c = tone.color;
    final tint = c.withValues(alpha: AppAlpha.badge);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: onImage ? Color.alphaBlend(tint, AppColors.bg.withValues(alpha: 0.85)) : tint,
        border: tone == Tone.neutral ? null : Border.all(color: c.withValues(alpha: AppAlpha.line)),
        borderRadius: BorderRadius.circular(AppRadius.badge),
      ),
      child: Text(text.toUpperCase(), style: AppText.badge.copyWith(color: c)),
    );
  }
}

/// "Alive/dead" dot (§ 12.2) — the only place with a glow. Pulses; faster on error.
class StatusDot extends StatefulWidget {
  final bool ok;
  const StatusDot({super.key, required this.ok});

  @override
  State<StatusDot> createState() => _StatusDotState();
}

class _StatusDotState extends State<StatusDot> with SingleTickerProviderStateMixin {
  late final _c = AnimationController(vsync: this, duration: _half)..repeat(reverse: true);

  Duration get _half => Duration(milliseconds: widget.ok ? 1000 : 750);

  @override
  void didUpdateWidget(StatusDot old) {
    super.didUpdateWidget(old);
    if (old.ok != widget.ok) {
      _c
        ..duration = _half
        ..repeat(reverse: true);
    }
  }

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final c = widget.ok ? AppColors.ok : AppColors.error;
    return FadeTransition(
      opacity: Tween<double>(begin: 1, end: 0.4).animate(_c),
      child: Container(
        width: 8,
        height: 8,
        decoration: BoxDecoration(
          color: c,
          shape: BoxShape.circle,
          boxShadow: [BoxShadow(color: c, blurRadius: 8)],
        ),
      ),
    );
  }
}

// ─── Cards ─────────────────────────────────────────────────────────────────

/// Card (§ 8.1): surface, border, radius 12, no shadow. The caller sets the padding.
class AppCard extends StatelessWidget {
  final Widget child;
  final EdgeInsetsGeometry? padding;
  const AppCard({super.key, required this.child, this.padding});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: padding,
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        color: AppColors.surface,
        border: Border.all(color: AppColors.border),
        borderRadius: BorderRadius.circular(AppRadius.card),
      ),
      child: child,
    );
  }
}

class NetImage extends StatelessWidget {
  final String? url;

  const NetImage(this.url, {super.key});

  @override
  Widget build(BuildContext context) {
    final fallback = Container(
      color: AppColors.surface,
      alignment: Alignment.center,
      child: const Icon(Icons.movie_outlined, color: AppColors.border, size: 32),
    );
    if (url == null) return fallback;
    return CachedNetworkImage(
      imageUrl: url!,
      fit: BoxFit.cover,
      fadeInDuration: const Duration(milliseconds: 150),
      placeholder: (_, _) => const ColoredBox(color: AppColors.surface),
      errorWidget: (_, _, _) => fallback,
    );
  }
}

/// Card that visibly reacts to remote focus: scales up slightly and gets a cyan border.
/// No glow — per the manifest, only status dots glow.
class FocusCard extends StatefulWidget {
  final Widget child;
  final VoidCallback onTap;
  final VoidCallback? onLongPress;
  final bool autofocus;
  final BorderRadius borderRadius;
  final double focusScale;

  const FocusCard({
    super.key,
    required this.child,
    required this.onTap,
    this.onLongPress,
    this.autofocus = false,
    this.borderRadius = const BorderRadius.all(Radius.circular(AppRadius.card)),
    this.focusScale = 1.07,
  });

  @override
  State<FocusCard> createState() => _FocusCardState();
}

class _FocusCardState extends State<FocusCard> {
  bool _focused = false;
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final lit = _focused || _hovered;
    return AnimatedScale(
      scale: _focused ? widget.focusScale : 1,
      duration: const Duration(milliseconds: 150),
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        // The border is painted over the content, so its focused width doesn't shift the layout.
        foregroundDecoration: BoxDecoration(
          borderRadius: widget.borderRadius,
          border: Border.all(
            color: _focused
                ? AppColors.accent
                : (lit ? AppColors.accent.withValues(alpha: AppAlpha.line) : AppColors.border),
            width: _focused ? 2 : 1,
          ),
        ),
        child: Material(
          color: AppColors.surface,
          borderRadius: widget.borderRadius,
          clipBehavior: Clip.antiAlias,
          child: InkWell(
            autofocus: widget.autofocus,
            onTap: widget.onTap,
            onLongPress: widget.onLongPress,
            onFocusChange: (f) => setState(() => _focused = f),
            onHover: (h) => setState(() => _hovered = h),
            focusColor: Colors.transparent,
            hoverColor: Colors.transparent,
            splashColor: Colors.transparent,
            highlightColor: Colors.transparent,
            child: widget.child,
          ),
        ),
      ),
    );
  }
}

// ─── Segmented control ─────────────────────────────────────────────────────

/// Pick one of 2–8 short options (§ 13.5). Each segment is focusable with a remote.
class Segmented<T> extends StatelessWidget {
  final List<(T, String)> options;
  final T value;
  final ValueChanged<T> onChanged;
  final String? semanticLabel;

  const Segmented({super.key, required this.options, required this.value, required this.onChanged, this.semanticLabel});

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: semanticLabel,
      container: true,
      child: Container(
        height: AppButtons.height,
        padding: const EdgeInsets.all(2),
        decoration: BoxDecoration(
          color: AppColors.bg,
          border: Border.all(color: AppColors.border),
          borderRadius: BorderRadius.circular(AppRadius.control),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            for (final (i, (v, label)) in options.indexed) ...[
              if (i > 0) const SizedBox(width: 2),
              _Segment(label: label, active: v == value, onTap: () => onChanged(v)),
            ],
          ],
        ),
      ),
    );
  }
}

class _Segment extends StatefulWidget {
  final String label;
  final bool active;
  final VoidCallback onTap;
  const _Segment({required this.label, required this.active, required this.onTap});

  @override
  State<_Segment> createState() => _SegmentState();
}

class _SegmentState extends State<_Segment> {
  bool _focused = false;

  @override
  Widget build(BuildContext context) {
    final color = widget.active || _focused ? AppColors.accent : AppColors.muted;
    const radius = BorderRadius.all(Radius.circular(AppRadius.segment));
    return Semantics(
      selected: widget.active,
      button: true,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        decoration: BoxDecoration(
          color: widget.active ? AppColors.accent.withValues(alpha: AppAlpha.segment) : Colors.transparent,
          borderRadius: radius,
        ),
        foregroundDecoration: BoxDecoration(
          borderRadius: radius,
          border: _focused ? Border.all(color: AppColors.accent, width: 1.5) : null,
        ),
        child: InkWell(
          borderRadius: radius,
          onTap: widget.onTap,
          onFocusChange: (f) => setState(() => _focused = f),
          focusColor: Colors.transparent,
          hoverColor: AppColors.accent.withValues(alpha: AppAlpha.tint),
          splashColor: Colors.transparent,
          highlightColor: Colors.transparent,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Center(
              widthFactor: 1,
              child: Text(widget.label, style: AppText.button.copyWith(color: color)),
            ),
          ),
        ),
      ),
    );
  }
}
