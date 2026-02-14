import { useEffect, useRef } from 'react'
import { AudioService } from '../../bindings/github.com/dannygim/meeting-transcriber/services'

interface SpectrumProps {
  active: boolean
}

const BAR_COUNT = 32
const POLL_INTERVAL = 60 // ms

export function Spectrum({ active }: SpectrumProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const animRef = useRef<number>(0)
  const dataRef = useRef<number[]>(new Array(BAR_COUNT).fill(0))
  const smoothRef = useRef<number[]>(new Array(BAR_COUNT).fill(0))

  useEffect(() => {
    if (!active) {
      smoothRef.current = new Array(BAR_COUNT).fill(0)
      dataRef.current = new Array(BAR_COUNT).fill(0)
      const canvas = canvasRef.current
      if (canvas) {
        const ctx = canvas.getContext('2d')
        if (ctx) ctx.clearRect(0, 0, canvas.width, canvas.height)
      }
      return
    }

    let polling = true

    const poll = async () => {
      while (polling) {
        try {
          const spectrum = await AudioService.GetSpectrum()
          if (spectrum && spectrum.length === BAR_COUNT) {
            dataRef.current = spectrum
          }
        } catch { /* ignore */ }
        await new Promise(r => setTimeout(r, POLL_INTERVAL))
      }
    }
    poll()

    const draw = () => {
      const canvas = canvasRef.current
      if (!canvas) return
      const ctx = canvas.getContext('2d')
      if (!ctx) return

      const dpr = window.devicePixelRatio || 1
      const rect = canvas.getBoundingClientRect()
      canvas.width = rect.width * dpr
      canvas.height = rect.height * dpr
      ctx.scale(dpr, dpr)

      const w = rect.width
      const h = rect.height
      ctx.clearRect(0, 0, w, h)

      const gap = 3
      const barWidth = (w - gap * (BAR_COUNT - 1)) / BAR_COUNT
      const data = dataRef.current
      const smooth = smoothRef.current
      const radius = Math.min(barWidth / 2, 3)

      for (let i = 0; i < BAR_COUNT; i++) {
        const target = data[i] || 0
        // Fast attack, slow decay
        if (target > smooth[i]) {
          smooth[i] = smooth[i] * 0.3 + target * 0.7
        } else {
          smooth[i] = smooth[i] * 0.85 + target * 0.15
        }

        const val = smooth[i]
        const barH = Math.max(2, val * h * 0.9)
        const x = i * (barWidth + gap)
        const y = h - barH

        // Color gradient: cyan → violet based on frequency position
        const t = i / (BAR_COUNT - 1)
        const hue = 190 - t * 110
        const sat = 75 + val * 25
        const lum = 45 + val * 20

        // Glow
        ctx.shadowColor = `hsla(${hue}, ${sat}%, ${lum}%, 0.6)`
        ctx.shadowBlur = val * 12

        ctx.fillStyle = `hsla(${hue}, ${sat}%, ${lum}%, 0.9)`
        ctx.beginPath()
        ctx.roundRect(x, y, barWidth, barH, radius)
        ctx.fill()
      }

      // Reset shadow for next frame
      ctx.shadowColor = 'transparent'
      ctx.shadowBlur = 0

      animRef.current = requestAnimationFrame(draw)
    }
    animRef.current = requestAnimationFrame(draw)

    return () => {
      polling = false
      cancelAnimationFrame(animRef.current)
    }
  }, [active])

  return (
    <canvas
      ref={canvasRef}
      className="spectrum-canvas"
    />
  )
}
