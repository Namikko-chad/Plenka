import { ref } from 'vue'
import { useRouter } from 'vue-router'
import type { Media, Release } from '@/api/types'
import { msg } from '@/i18n'
import { useToast } from '@/stores/toast'
import { useTorrents } from '@/stores/torrents'
import { errorText } from '@/utils/errors'
import { toMedia } from '@/utils/media'
import { dedupKey } from '@/utils/releases'

/** Release tap: add to the torrent engine, then open its file list. */
export function useAddRelease() {
  const router = useRouter()
  const torrents = useTorrents()
  const toast = useToast()
  const busyKey = ref<string | null>(null)

  async function add(r: Release, media: Media | null) {
    if (busyKey.value) return
    busyKey.value = dedupKey(r)
    try {
      const st = await torrents.add({
        link: r.link || r.magnet,
        title: media?.title || r.title,
        poster: media?.posterPath ? `${location.origin}/img/t/p/w342${media.posterPath}` : '',
        data: media ? JSON.stringify(toMedia(media)) : '',
      })
      const hash = st.hash || r.infoHash
      if (!hash) throw new Error(msg.files.noHash)
      await router.push({ name: 'files', params: { hash } })
    } catch (e) {
      toast.error(errorText(e))
    } finally {
      busyKey.value = null
    }
  }

  return { busyKey, add }
}
