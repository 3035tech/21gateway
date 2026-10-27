import * as Select from '@radix-ui/react-select'
import { Check, ChevronDown, ChevronUp } from 'lucide-react'
import { useCallback, useState } from 'react'

type Option = { value: string | number; label: string; disabled?: boolean }

const EMPTY_VALUE = '__21mobile_empty__'

export function AppSelect({
  value,
  onValueChange,
  options,
  label,
  className = '',
  disabled = false,
}: {
  value: string | number
  onValueChange: (value: string) => void
  options: Option[]
  label: string
  className?: string
  disabled?: boolean
}) {
  const selected = String(value) || EMPTY_VALUE
  const [portalContainer, setPortalContainer] = useState<HTMLElement>()
  const setTrigger = useCallback((node: HTMLButtonElement | null) => {
    setPortalContainer(node?.closest('dialog') ?? undefined)
  }, [])
  return (
    <Select.Root
      value={selected}
      onValueChange={(next) => onValueChange(next === EMPTY_VALUE ? '' : next)}
      disabled={disabled}
    >
      <Select.Trigger
        ref={setTrigger}
        aria-label={label}
        className={`form-control select-trigger ${className}`}
      >
        <Select.Value />
        <Select.Icon
          aria-hidden="true"
          className="ml-3 shrink-0 text-brand-purple"
        >
          <ChevronDown size={17} strokeWidth={2.25} />
        </Select.Icon>
      </Select.Trigger>
      <Select.Portal container={portalContainer}>
        <Select.Content
          position="popper"
          sideOffset={5}
          collisionPadding={8}
          className="select-content"
        >
          <Select.ScrollUpButton className="select-scroll-button">
            <ChevronUp size={16} />
          </Select.ScrollUpButton>
          <Select.Viewport className="select-viewport">
            {options.map((option) => (
              <Select.Item
                key={String(option.value)}
                value={String(option.value) || EMPTY_VALUE}
                disabled={option.disabled}
                className="select-item"
              >
                <Select.ItemText>{option.label}</Select.ItemText>
                <Select.ItemIndicator className="ml-auto text-brand-purple">
                  <Check size={16} strokeWidth={2.5} />
                </Select.ItemIndicator>
              </Select.Item>
            ))}
          </Select.Viewport>
          <Select.ScrollDownButton className="select-scroll-button">
            <ChevronDown size={16} />
          </Select.ScrollDownButton>
        </Select.Content>
      </Select.Portal>
    </Select.Root>
  )
}
