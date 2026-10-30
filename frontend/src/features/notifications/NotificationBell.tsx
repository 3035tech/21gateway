import { useState } from 'react'
import { Bell } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { listNotifications, markNotificationRead } from './api'

export function NotificationBell() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
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
    <div className="relative">
      <button
        type="button"
        aria-label={t('notifications.bellLabel', { count })}
        aria-expanded={open}
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
          <ul className="max-h-96 overflow-y-auto">
            {query.data?.items.map((item) => (
              <li
                key={item.id}
                className={`border-b border-slate-100 p-2 text-sm last:border-0 ${item.readAt ? 'text-slate-600' : 'bg-purple-50 text-slate-900'}`}
              >
                <Link
                  to={item.path}
                  onClick={() => {
                    if (!item.readAt) read.mutate(item.id)
                    setOpen(false)
                  }}
                  className="block rounded-md p-1 hover:bg-purple-100"
                >
                  <span className="font-semibold">{item.title}</span>
                  <span className="mt-1 block text-xs leading-5">
                    {item.description}
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
