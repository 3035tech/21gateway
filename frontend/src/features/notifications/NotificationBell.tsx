import { useEffect, useRef, useState } from 'react'
import { Bell } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { listNotifications, markNotificationRead } from './api'
import { notificationDescription, notificationTitle } from './display'

export function NotificationBell() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    if (!open) return
    const onPointerDown = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false)
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false)
        triggerRef.current?.focus()
      }
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])
  const client = useQueryClient()
  const query = useQuery({
    queryKey: ['admin', 'notifications', 'bell'],
    queryFn: ({ signal }) =>
      listNotifications(
        {
          page: 1,
          pageSize: 5,
          unread: false,
          sort: 'openedAt',
          order: 'desc',
        },
        signal,
      ),
    refetchInterval: 60_000,
  })
  const read = useMutation({
    mutationFn: markNotificationRead,
    onSuccess: () =>
      client.invalidateQueries({ queryKey: ['admin', 'notifications'] }),
  })
  const count = query.data?.unreadCount ?? 0
  return (
    <div ref={rootRef} className="relative">
      <button
        ref={triggerRef}
        type="button"
        aria-label={t('notifications.bellLabel', { count })}
        aria-expanded={open}
        aria-controls="notification-bell-panel"
        onClick={() => setOpen((value) => !value)}
        className="relative flex size-10 items-center justify-center rounded-xl text-slate-600 hover:bg-purple-50 hover:text-brand-purple focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-purple"
      >
        <Bell size={20} aria-hidden="true" />
        {count > 0 ? (
          <span className="absolute -right-1 -top-1 min-w-5 rounded-full bg-rose-600 px-1 text-center text-[11px] font-bold leading-5 text-white">
            {count > 99 ? '99+' : count}
          </span>
        ) : null}
      </button>
      {open ? (
        <div
          id="notification-bell-panel"
          className="absolute right-0 z-30 mt-2 w-[min(22rem,calc(100vw-2rem))] rounded-2xl border border-slate-200 bg-white p-3 shadow-xl"
          role="region"
          aria-label={t('notifications.title')}
        >
          <div className="flex items-center justify-between gap-2 border-b border-slate-100 px-1 pb-2">
            <strong className="text-sm text-slate-900">
              {t('notifications.title')}
            </strong>
            <Link
              to="/notificacoes"
              onClick={() => setOpen(false)}
              className="text-xs font-semibold text-brand-purple underline"
            >
              {t('notifications.viewAll')}
            </Link>
          </div>
          {query.isPending ? (
            <p className="p-3 text-sm text-slate-600">{t('common.loading')}</p>
          ) : null}
          {query.isError ? (
            <p role="alert" className="p-3 text-sm text-rose-700">
              {t('notifications.loadFailed')}
            </p>
          ) : null}
          {query.data?.items.length === 0 ? (
            <p className="p-3 text-sm text-slate-600">
              {t('notifications.empty')}
            </p>
          ) : null}
          <ul className="max-h-96 space-y-1 overflow-y-auto pt-2">
            {query.data?.items.map((item) => (
              <li
                key={item.id}
                className={`rounded-lg p-2 text-sm ${item.readAt ? 'text-slate-600' : 'bg-purple-50 text-slate-900'}`}
              >
                <Link
                  to={item.path}
                  onClick={() => {
                    if (!item.readAt) read.mutate(item.id)
                    setOpen(false)
                  }}
                  className="block rounded-md p-1 hover:bg-purple-100 focus-visible:outline-2 focus-visible:outline-brand-purple"
                >
                  <span className="flex items-center gap-2 font-semibold">
                    {!item.readAt ? (
                      <span
                        className="size-2 shrink-0 rounded-full bg-brand-purple"
                        aria-label={t('notifications.unread')}
                      />
                    ) : null}
                    {notificationTitle(t, item.type, item.title)}
                  </span>
                  <span className="mt-1 block text-xs leading-5">
                    {notificationDescription(t, item.type, item.description)}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  )
}
