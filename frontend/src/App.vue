<script setup lang="ts">
import AppShell from '@/components/layout/AppShell.vue'
import ExternalSheet from '@/components/media/ExternalSheet.vue'
import PlayChooser from '@/components/media/PlayChooser.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import ToastHost from '@/components/ui/ToastHost.vue'
import { useServer } from '@/stores/server'

void useServer().loadInfo()
</script>

<template>
  <AppShell />
  <ExternalSheet />
  <PlayChooser />
  <ConfirmDialog />
  <!-- after the dialogs: toasts must never sit under a modal overlay -->
  <ToastHost />
</template>
