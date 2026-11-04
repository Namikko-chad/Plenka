<script setup lang="ts">
import { computed } from 'vue'
import { msg } from '@/i18n'
import { errorText } from '@/utils/errors'
import Icon from './Icon.vue'
import UiButton from './UiButton.vue'

const props = defineProps<{ error: unknown; retry?: boolean }>()
defineEmits<{ retry: [] }>()
const text = computed(() => (typeof props.error === 'string' ? props.error : errorText(props.error)))
</script>

<template>
  <div class="banner" role="alert">
    <Icon name="alert" :size="18" class="banner__icon" />
    <span class="banner__text">{{ text }}</span>
    <UiButton v-if="retry" variant="red" icon="refresh" @click="$emit('retry')">{{ msg.common.retry }}</UiButton>
  </div>
</template>

<style scoped>
.banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--r-control);
  background: rgba(var(--rgb-error), 0.07);
  border: 1px solid rgba(var(--rgb-error), var(--a-line));
  color: var(--c-error);
  font-size: 12px;
}
.banner__icon {
  flex-shrink: 0;
}
.banner__text {
  flex: 1;
  line-height: 1.4;
}
</style>
