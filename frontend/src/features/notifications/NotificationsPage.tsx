import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  listNotifications,
  listNotificationPreferences,
  markNotificationRead,
  setNotificationPreference,
} from './api'

export function NotificationsPage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [unread, setUnread] = useState(false)
  const [sort, setSort] = useState<'openedAt' | 'title'>('openedAt')
  const [order, setOrder] = useState<'asc' | 'desc'>('desc')
  const list = useQuery({
    queryKey: [
      'admin',
      'notifications',
      'list',
      page,
      pageSize,
      unread,
      sort,
      order,
    ],
    queryFn: ({ signal }) =>
      listNotifications({ page, pageSize, unread, sort, order }, signal),
    refetchInterval: 60_000,
  })
  const preferences = useQuery({
    queryKey: ['admin', 'notification-preferences'],
    queryFn: ({ signal }) => listNotificationPreferences(signal),
  })
  const read = useMutation({
    mutationFn: markNotificationRead,
    onSuccess: () =>
      client.invalidateQueries({ queryKey: ['admin', 'notifications'] }),
  })
  const save = useMutation({
    mutationFn: ({ code, enabled }: { code: string; enabled: boolean }) =>
      setNotificationPreference(code, enabled),
    onSuccess: () =>
      client.invalidateQueries({
        queryKey: ['admin', 'notification-preferences'],
      }),
  })
  const changeSort = (field: 'openedAt' | 'title') => {
    setPage(1)
    if (sort === field) setOrder((value) => (value === 'asc' ? 'desc' : 'asc'))
    else {
      setSort(field)
      setOrder(field === 'openedAt' ? 'desc' : 'asc')
    }
  }
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">
          {t('notifications.title')}
        </h1>
        <p className="mt-1 text-sm text-slate-600">
          {t('notifications.intro')}
        </p>
      </div>
      <section
        className="rounded-2xl border border-slate-200 bg-white p-5"
        aria-label={t('notifications.emailSettings')}
      >
        <h2 className="text-lg font-bold text-slate-900">
          {t('notifications.emailSettings')}
        </h2>
        <p className="mt-1 text-sm text-slate-600">
          {t('notifications.emailHint')}
        </p>
        {preferences.data && !preferences.data.emailAvailable ? (
          <p role="status" className="mt-3 text-sm text-amber-800">
            {t('notifications.smtpUnavailable')}
          </p>
        ) : null}
        {preferences.isError ? (
          <p role="alert" className="mt-3 text-sm text-rose-700">
            {t('notifications.loadFailed')}
          </p>
        ) : null}
        {save.isError ? (
          <p role="alert" className="mt-3 text-sm text-rose-700">
            {t('notifications.saveFailed')}
          </p>
        ) : null}
        <div className="mt-4 divide-y divide-slate-100">
          {preferences.data?.items.map((preference) => (
            <label
              key={preference.code}
              className="flex min-h-14 cursor-pointer items-center justify-between gap-4 py-3 text-sm text-slate-800"
            >
              <span>{preference.title}</span>
              <span className="flex shrink-0 items-center gap-2 font-semibold text-brand-purple">
                {t('notifications.sendEmail')}
                <input
                  type="checkbox"
                  checked={preference.emailEnabled}
                  disabled={save.isPending || !preferences.data.emailAvailable}
                  onChange={(event) =>
                    save.mutate({
                      code: preference.code,
                      enabled: event.target.checked,
                    })
                  }
                  className="size-5 accent-brand-purple"
                />
              </span>
            </label>
          ))}
        </div>
      </section>
      <section
        className="rounded-2xl border border-slate-200 bg-white p-5"
        aria-label={t('notifications.history')}
      >
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-lg font-bold text-slate-900">
            {t('notifications.history')}
          </h2>
          <div className="flex flex-wrap items-center gap-3 text-sm">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={unread}
                onChange={(event) => {
                  setUnread(event.target.checked)
                  setPage(1)
                }}
              />
              {t('notifications.unreadOnly')}
            </label>
            <label className="flex items-center gap-2">
              {t('common.itemsPerPage')}
              <select
                value={pageSize}
                onChange={(event) => {
                  setPageSize(Number(event.target.value))
                  setPage(1)
                }}
                className="rounded-lg border border-slate-300 px-2 py-1"
              >
                <option>10</option>
                <option>20</option>
                <option>50</option>
              </select>
            </label>
          </div>
        </div>
        {list.isError ? (
          <p role="alert" className="mt-4 text-sm text-rose-700">
            {t('notifications.loadFailed')}
          </p>
        ) : null}
        {list.isPending ? (
          <p className="mt-4 text-sm text-slate-600">{t('common.loading')}</p>
        ) : null}
        {list.data?.items.length === 0 ? (
          <p className="mt-4 text-sm text-slate-600">
            {t('notifications.empty')}
          </p>
        ) : null}
        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-slate-200 text-slate-600">
                <th className="p-2">
                  <button
                    onClick={() => changeSort('openedAt')}
                    className="font-semibold hover:underline"
                  >
                    {t('notifications.date')}{' '}
                    {sort === 'openedAt' ? (order === 'asc' ? '↑' : '↓') : ''}
                  </button>
                </th>
                <th className="p-2">
                  <button
                    onClick={() => changeSort('title')}
                    className="font-semibold hover:underline"
                  >
                    {t('notifications.type')}{' '}
                    {sort === 'title' ? (order === 'asc' ? '↑' : '↓') : ''}
                  </button>
                </th>
                <th className="p-2">{t('notifications.description')}</th>
                <th className="p-2">{t('common.status')}</th>
                <th className="p-2">{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {list.data?.items.map((item) => (
                <tr
                  key={item.id}
                  className="border-b border-slate-100 align-top"
                >
                  <td className="whitespace-nowrap p-2">
                    {new Date(item.openedAt).toLocaleString()}
                  </td>
                  <td className="p-2 font-semibold">{item.title}</td>
                  <td className="max-w-md p-2">{item.description}</td>
                  <td className="p-2">
                    {item.resolvedAt
                      ? t('notifications.resolved')
                      : t('notifications.active')}
                  </td>
                  <td className="whitespace-nowrap p-2">
                    <Link
                      className="text-brand-purple underline"
                      to={item.path}
                      onClick={() => {
                        if (!item.readAt) read.mutate(item.id)
                      }}
                    >
                      {t('common.view')}
                    </Link>
                    {!item.readAt ? (
                      <button
                        className="ml-3 text-brand-purple underline"
                        onClick={() => read.mutate(item.id)}
                      >
                        {t('notifications.markRead')}
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {list.data ? (
          <div className="mt-4 flex items-center justify-between gap-3 text-sm">
            <span>
              {t('common.resultsRange', {
                from: list.data.total ? (page - 1) * pageSize + 1 : 0,
                to: Math.min(page * pageSize, list.data.total),
                total: list.data.total,
              })}
            </span>
            <div className="flex gap-2">
              <button
                disabled={page === 1}
                onClick={() => setPage(page - 1)}
                className="rounded-lg border px-3 py-2 disabled:opacity-40"
              >
                {t('common.previous')}
              </button>
              <button
                disabled={page * pageSize >= list.data.total}
                onClick={() => setPage(page + 1)}
                className="rounded-lg border px-3 py-2 disabled:opacity-40"
              >
                {t('common.next')}
              </button>
            </div>
          </div>
        ) : null}
      </section>
    </div>
  )
}
