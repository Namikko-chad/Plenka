<script setup lang="ts">
import { msg } from '@/i18n'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { CacheState } from '@/api/types'

/** Compact cache map on a canvas: one cell per bucket of pieces. */
const props = defineProps<{ cache: CacheState }>()
const wrap = ref<HTMLElement | null>(null)
const canvas = ref<HTMLCanvasElement | null>(null)
let ro: ResizeObserver | null = null

const CELL = 6
const GAP = 1

function css(name: string) {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

function draw() {
  const el = canvas.value
  const box = wrap.value
  if (!el || !box) return
  const c = props.cache
  const count = Math.max(0, c.piecesCount || 0)
  const width = box.clientWidth
  const cols = Math.max(1, Math.floor((width + GAP) / (CELL + GAP)))
  const maxCells = cols * 24
  const cells = Math.min(count, maxCells)
  const per = cells > 0 ? count / cells : 1
  const rows = Math.max(1, Math.ceil(cells / cols))
  const dpr = window.devicePixelRatio || 1
  const h = rows * (CELL + GAP)
  el.width = Math.round(width * dpr)
  el.height = Math.round(h * dpr)
  el.style.height = `${h}px`
  const ctx = el.getContext('2d')
  if (!ctx) return
  ctx.scale(dpr, dpr)

  // per-cell fill fraction and flags
  const fill = new Float32Array(cells)
  const prio = new Uint8Array(cells)
  const reading = new Uint8Array(cells)
  for (const p of Object.values(c.pieces)) {
    const i = Math.min(cells - 1, Math.floor(p.id / per))
    if (i < 0) continue
    const frac = p.completed ? 1 : p.length > 0 ? p.size / p.length : 0
    fill[i] = fill[i]! + frac / per
    if (p.priority > 0 && !p.completed) prio[i] = 1
  }
  for (const r of c.readers) {
    for (let id = r.start; id <= r.end && id < count; id += Math.max(1, Math.floor(per))) {
      reading[Math.min(cells - 1, Math.floor(id / per))] = 1
    }
  }

  const accent = css('--c-accent')
  const border = css('--c-border')
  const warning = css('--c-warning')
  const purple = css('--c-purple-l')
  for (let i = 0; i < cells; i++) {
    const x = (i % cols) * (CELL + GAP)
    const y = Math.floor(i / cols) * (CELL + GAP)
    const f = Math.min(1, fill[i]!)
    ctx.globalAlpha = 1
    ctx.fillStyle = border
    ctx.fillRect(x, y, CELL, CELL)
    if (f > 0) {
      ctx.fillStyle = accent
      ctx.globalAlpha = f >= 0.999 ? 1 : 0.25 + 0.6 * f
      ctx.fillRect(x, y, CELL, CELL)
    } else if (prio[i]) {
      ctx.fillStyle = warning
      ctx.globalAlpha = 0.7
      ctx.fillRect(x, y, CELL, CELL)
    }
    if (reading[i]) {
      ctx.globalAlpha = 1
      ctx.strokeStyle = purple
      ctx.lineWidth = 1
      ctx.strokeRect(x + 0.5, y + 0.5, CELL - 1, CELL - 1)
    }
  }
  ctx.globalAlpha = 1
}

onMounted(() => {
  draw()
  ro = new ResizeObserver(() => draw())
  if (wrap.value) ro.observe(wrap.value)
})
onBeforeUnmount(() => ro?.disconnect())
watch(() => props.cache, draw)
</script>

<template>
  <div ref="wrap" class="pmap">
    <canvas ref="canvas" class="pmap__canvas" role="img" :aria-label="msg.torrents.cacheMap" />
  </div>
</template>

<style scoped>
.pmap {
  width: 100%;
}
.pmap__canvas {
  display: block;
  width: 100%;
}
</style>
