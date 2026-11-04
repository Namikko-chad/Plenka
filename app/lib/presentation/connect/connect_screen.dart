import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../core/config/app_config.dart';
import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../common/widgets/common.dart';
import 'connection_cubit.dart';
import '../../l10n/l10n.dart';

/// Shown until the server is found. Usually the user sees it for a split second.
class ConnectScreen extends StatelessWidget {
  const ConnectScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: PageBackground(
        child: BlocBuilder<ConnectionCubit, ServerConnectionState>(
          builder: (context, state) {
            if (state.status != ConnectionStatus.notFound) {
              final address = state.address;
              return LoadingIndicator(
                message: address == null ? context.l10n.connectSearching : context.l10n.connectConnectingTo(address),
              );
            }
            return _NotFound(saved: state.address);
          },
        ),
      ),
    );
  }
}

class _NotFound extends StatefulWidget {
  /// The manually set server that didn't respond; null — auto-discovery found nothing.
  final String? saved;
  const _NotFound({this.saved});

  @override
  State<_NotFound> createState() => _NotFoundState();
}

class _NotFoundState extends State<_NotFound> {
  late final _controller = TextEditingController(text: widget.saved ?? '');
  bool _busy = false;
  String? _error;

  Future<void> _connect() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    final ok = await context.read<ConnectionCubit>().connectTo(_controller.text);
    if (!mounted) return;
    setState(() {
      _busy = false;
      _error = ok ? null : context.l10n.connectNoResponse;
    });
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final saved = widget.saved;
    final cubit = context.read<ConnectionCubit>();
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 480),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Icon(Icons.router_outlined, size: 32, color: AppColors.muted),
              const SizedBox(height: 12),
              Text(l.connectNotFound, textAlign: TextAlign.center, style: AppText.pageTitle),
              const SizedBox(height: 8),
              Text(
                saved == null ? l.connectNotFoundHint : l.connectSavedNotResponding(saved),
                textAlign: TextAlign.center,
                style: AppText.body.copyWith(color: AppColors.muted),
              ),
              const SizedBox(height: 24),
              FilledButton.icon(
                autofocus: true,
                onPressed: _busy ? null : cubit.connect,
                icon: const Icon(Icons.refresh),
                label: Text(saved == null ? l.connectSearchAgain : l.retry),
              ),
              if (saved != null) ...[
                const SizedBox(height: 12),
                OutlinedButton.icon(
                  onPressed: _busy ? null : cubit.discoverAutomatically,
                  icon: const Icon(Icons.wifi_find),
                  label: Text(l.connectSearchAuto),
                ),
              ],
              const SizedBox(height: 28),
              FieldLabel(l.routerAddress),
              TextField(
                controller: _controller,
                keyboardType: TextInputType.url,
                textInputAction: TextInputAction.go,
                onSubmitted: (_) => _connect(),
                decoration: InputDecoration(hintText: l.connectAddressHint(AppConfig.serverPort), errorText: _error),
              ),
              const SizedBox(height: 12),
              OutlinedButton(
                onPressed: _busy ? null : _connect,
                child: _busy
                    ? const IconTheme(
                        data: IconThemeData(color: AppColors.accent),
                        child: Spinner(size: 14),
                      )
                    : Text(l.connectButton),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
