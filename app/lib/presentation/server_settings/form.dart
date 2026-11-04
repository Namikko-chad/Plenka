import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../core/theme/app_theme.dart';
import '../../core/theme/colors.dart';
import '../../l10n/l10n.dart';
import '../common/widgets/common.dart';

/// Server settings screen: title, loading/error and a form column of limited width
/// (full-width fields read poorly on a TV).
class SettingsPage extends StatelessWidget {
  final String title;
  final bool loading;
  final Object? error;
  final VoidCallback onRetry;
  final List<Widget> Function() children;

  const SettingsPage({
    super.key,
    required this.title,
    required this.loading,
    required this.error,
    required this.onRetry,
    required this.children,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: PageBackground(
        child: Builder(
          builder: (context) {
            if (error != null) return MessageView.error(error!, onRetry: onRetry);
            if (loading) return const LoadingIndicator();
            return Align(
              alignment: Alignment.topCenter,
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 760),
                child: ListView(padding: pagePadding(context).copyWith(top: 8, bottom: 48), children: children()),
              ),
            );
          },
        ),
      ),
    );
  }
}

/// Input field with a label above and a hint below.
class LabeledField extends StatelessWidget {
  final String label;
  final TextEditingController controller;
  final String? hint;
  final String? helper;
  final bool number;
  final bool obscure;
  final bool autofocus;
  final TextInputType? keyboardType;

  const LabeledField({
    super.key,
    required this.label,
    required this.controller,
    this.hint,
    this.helper,
    this.number = false,
    this.obscure = false,
    this.autofocus = false,
    this.keyboardType,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          FieldLabel(label),
          TextField(
            controller: controller,
            autofocus: autofocus,
            obscureText: obscure,
            keyboardType: number ? TextInputType.number : keyboardType,
            inputFormatters: number ? [FilteringTextInputFormatter.digitsOnly] : null,
            decoration: InputDecoration(hintText: hint),
          ),
          if (helper != null) ...[const SizedBox(height: 6), Text(helper!, style: AppText.small)],
        ],
      ),
    );
  }
}

/// Switch as a row (the whole row is remote-focusable, OK toggles it).
class SwitchRow extends StatelessWidget {
  final String label;
  final bool value;
  final ValueChanged<bool> onChanged;
  final bool autofocus;

  const SwitchRow({
    super.key,
    required this.label,
    required this.value,
    required this.onChanged,
    this.autofocus = false,
  });

  @override
  Widget build(BuildContext context) {
    return SwitchListTile(
      autofocus: autofocus,
      contentPadding: const EdgeInsets.symmetric(horizontal: 4),
      title: Text(label, style: AppText.body),
      value: value,
      onChanged: onChanged,
    );
  }
}

/// Button with a spinner while the action runs.
class BusyButton extends StatelessWidget {
  final String label;
  final IconData icon;
  final bool busy;
  final VoidCallback? onPressed;
  final ButtonVariant variant;

  const BusyButton({
    super.key,
    required this.label,
    required this.icon,
    required this.busy,
    required this.onPressed,
    this.variant = ButtonVariant.primary,
  });

  @override
  Widget build(BuildContext context) {
    return OutlinedButton.icon(
      style: AppButtons.of(variant),
      onPressed: busy ? null : onPressed,
      icon: busy
          ? const IconTheme(
              data: IconThemeData(color: AppColors.accent),
              child: Spinner(size: 14),
            )
          : Icon(icon, size: 16),
      label: Text(label),
    );
  }
}

Future<bool> confirmDialog(BuildContext context, {required String title, String? text, required String action}) async {
  final ok = await showDialog<bool>(
    context: context,
    builder: (ctx) => AlertDialog(
      title: Text(title),
      content: text == null ? null : Text(text),
      actions: [
        TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(ctx.l10n.cancel)),
        OutlinedButton(
          autofocus: true,
          style: AppButtons.of(ButtonVariant.red),
          onPressed: () => Navigator.pop(ctx, true),
          child: Text(action),
        ),
      ],
    ),
  );
  return ok == true;
}

void showMessage(BuildContext context, String text) =>
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(text)));
