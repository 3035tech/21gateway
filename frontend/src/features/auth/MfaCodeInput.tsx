import { useEffect, useRef } from 'react'

const digitsCount = 6

export function MfaCodeInput({
  value,
  onChange,
  onComplete,
  disabled = false,
}: {
  value: string
  onChange: (value: string) => void
  onComplete: (value: string) => void
  disabled?: boolean
}) {
  const inputs = useRef<Array<HTMLInputElement | null>>([])
  useEffect(() => {
    inputs.current[0]?.focus()
  }, [])
  const digits = value.padEnd(digitsCount, '_').slice(0, digitsCount).split('')
  function update(index: number, digit: string) {
    const next = [...digits]
    next[index] = digit || '_'
    const code = next.join('')
    onChange(code)
    if (digit && index < digitsCount - 1) inputs.current[index + 1]?.focus()
    if (/^\d{6}$/.test(code) && code !== value) onComplete(code)
  }
  function paste(text: string) {
    const code = text.replace(/\D/g, '').slice(0, digitsCount)
    if (!code) return
    onChange(code.padEnd(digitsCount, '_'))
    inputs.current[Math.min(code.length, digitsCount - 1)]?.focus()
    if (code.length === digitsCount && code !== value) onComplete(code)
  }
  return (
    <div
      role="group"
      aria-label="Código de segurança de seis dígitos"
      className="flex gap-2"
    >
      {digits.map((digit, index) => (
        <input
          key={index}
          ref={(element) => {
            inputs.current[index] = element
          }}
          aria-label={`Dígito ${index + 1} de ${digitsCount}`}
          className="form-control h-12 w-11 min-w-0 text-center text-xl font-bold tabular-nums sm:w-12"
          type="text"
          inputMode="numeric"
          pattern="[0-9]*"
          autoComplete={index === 0 ? 'one-time-code' : 'off'}
          maxLength={1}
          value={digit === '_' ? '' : digit}
          disabled={disabled}
          onFocus={(event) => event.currentTarget.select()}
          onChange={(event) => {
            const raw = event.target.value
            if (raw.length > 1) {
              paste(raw)
              return
            }
            if (/^\d?$/.test(raw)) update(index, raw)
          }}
          onPaste={(event) => {
            event.preventDefault()
            paste(event.clipboardData.getData('text'))
          }}
          onKeyDown={(event) => {
            if (event.key === 'Backspace') {
              event.preventDefault()
              if (digit !== '_') update(index, '')
              else if (index > 0) {
                update(index - 1, '')
                inputs.current[index - 1]?.focus()
              }
            } else if (event.key === 'ArrowLeft' && index > 0) {
              event.preventDefault()
              inputs.current[index - 1]?.focus()
            } else if (event.key === 'ArrowRight' && index < digitsCount - 1) {
              event.preventDefault()
              inputs.current[index + 1]?.focus()
            }
          }}
        />
      ))}
    </div>
  )
}
