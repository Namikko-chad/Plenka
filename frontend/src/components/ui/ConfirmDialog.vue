<script setup lang="ts">
import { computed } from 'vue'
import { msg } from '@/i18n'
import { useDialog } from '@/stores/dialog'
import Sheet from './Sheet.vue'
import UiButton from './UiButton.vue'

const dialog = useDialog()
const open = computed({
  get: () => !!dialog.current,
  set: (v) => {
    if (!v) dialog.answer(false)
  },
})
</script>

<template>
  <Sheet v-model="open" dialog :title="dialog.current?.title">
    <p v-if="dialog.current?.text" class="confirm-text">{{ dialog.current.text }}</p>
    <template #footer>
      <UiButton variant="neutral" @click="dialog.answer(false)">{{ msg.common.cancel }}</UiButton>
      <UiButton :variant="dialog.current?.danger ? 'red' : 'primary'" autofocus @click="dialog.answer(true)">
        {{ dialog.current?.confirmLabel ?? msg.common.ok }}
      </UiButton>
    </template>
  </Sheet>
</template>

<style scoped>
.confirm-text {
  margin: 0;
  color: var(--c-muted);
  line-height: 1.5;
}
</style>
