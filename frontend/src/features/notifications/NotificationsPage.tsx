import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Bell, CheckCheck, Eye, Mail } from 'lucide-react'
import { PageTabs, PageTabPanel } from '../../components/admin/PageTabs'
import {
  TableIconButton,
  TableIconLink,
} from '../../components/admin/TableIconAction'
import type { NotificationPreference } from './api'
import {
  listNotifications,
  listNotificationPreferences,
  markNotificationRead,
  setNotificationPreference,
} from './api'

export function NotificationsPage() {
  const { t, i18n } = useTranslation()
  const client = useQueryClient()
  const [activeTab, setActiveTab] = useState<'history' | 'email'>('history')
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
    enabled: activeTab === 'history',
    refetchInterval: activeTab === 'history' ? 60_000 : false,
  })
  const preferences = useQuery({
    queryKey: ['admin', 'notification-preferences'],
    queryFn: ({ signal }) => listNotificationPreferences(signal),
    enabled: activeTab === 'email',
  })
  const read = useMutation({
    mutationFn: markNotificationRead,
    onSuccess: () =>
      client.invalidateQueries({ queryKey: ['admin', 'notifications'] }),
  })
  const save = useMutation({
    mutationFn: ({ code, enabled }: { code: string; enabled: boolean }) =>
      setNotificationPreference(code, enabled),
    onSuccess: (saved) => {
      client.setQueryData<{
        items: NotificationPreference[]
        emailAvailable: boolean
      }>(['admin', 'notification-preferences'], (current) =>
        current
          ? {
              ...current,
              items: current.items.map((item) =>
                item.code === saved.code ? saved : item,
              ),
            }
          : current,
      )
      client.invalidateQueries({
        queryKey: ['admin', 'notification-preferences'],
      })
    },
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
      <PageTabs
        idPrefix="notifications"
        label={t('notifications.title')}
        tabs={[
          { id: 'history', label: t('notifications.occurrences'), icon: Bell },
          { id: 'email', label: t('notifications.emailSettings'), icon: Mail },
        ]}
        active={activeTab}
        onChange={setActiveTab}
      />
      <PageTabPanel
        idPrefix="notifications"
        tab="email"
        active={activeTab === 'email'}
      >
        <section
          className="overflow-hidden rounded-2xl border border-slate-200 bg-white"
          aria-label={t('notifications.emailSettings')}
        >
          <div className="border-b border-slate-100 px-5 py-5 sm:px-6">
            <h2 className="text-lg font-bold text-slate-900">
              {t('notifications.emailSettings')}
            </h2>
            <p className="mt-1 max-w-3xl text-sm leading-6 text-slate-600">
              {t('notifications.emailHint')}
            </p>
          </div>
          {preferences.data && !preferences.data.emailAvailable ? (
            <p
              role="status"
              className="mx-5 mt-4 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-900 sm:mx-6"
            >
              {t('notifications.smtpUnavailable')}
            </p>
          ) : null}
          {preferences.isError ? (
            <p role="alert" className="mx-5 mt-4 text-sm text-rose-700 sm:mx-6">
              {t('notifications.loadFailed')}
            </p>
          ) : null}
          {preferences.isPending ? (
            <p className="px-5 py-6 text-sm text-slate-600 sm:px-6">
              {t('common.loading')}
            </p>
          ) : null}
          <div className="divide-y divide-slate-100">
            {preferences.data?.items.map((preference) => (
              <label
                key={preference.code}
                className="flex min-h-20 cursor-pointer items-center justify-between gap-4 px-5 py-4 text-sm text-slate-800 hover:bg-slate-50 sm:px-6"
              >
                <span className="flex min-w-0 items-center gap-3">
                  <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-purple-50 text-brand-purple">
                    <Bell size={17} aria-hidden="true" />
                  </span>
                  <span className="min-w-0">
                    <span className="block font-semibold text-slate-900">
                      {preference.title}
                    </span>
                    <span className="mt-0.5 block text-xs text-slate-500">
                      {t('notifications.inAppAlways')}
                    </span>
                  </span>
                </span>
                <span className="flex shrink-0 flex-col items-end gap-1">
                  <span className="flex items-center gap-3 font-medium text-slate-600">
                    <span className="inline-flex items-center gap-1.5">
                      <Mail size={15} aria-hidden="true" />
                      <span className="hidden sm:inline">
                        {t('notifications.sendEmail')}
                      </span>
                    </span>
                    <input
                      type="checkbox"
                      checked={preference.emailEnabled}
                      disabled={
                        save.isPending ||
                        (!preferences.data.emailAvailable &&
                          !preference.emailEnabled)
                      }
                      onChange={(event) =>
                        save.mutate({
                          code: preference.code,
                          enabled: event.target.checked,
                        })
                      }
                      aria-label={`${preference.title}: ${t('notifications.sendEmail')}`}
                      className="peer sr-only"
                    />
                    <span
                      aria-hidden="true"
                      className="relative h-7 w-12 rounded-full bg-slate-300 transition-colors before:absolute before:left-1 before:top-1 before:size-5 before:rounded-full before:bg-white before:shadow-sm before:transition-transform peer-checked:bg-brand-purple peer-checked:before:translate-x-5 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-brand-purple peer-disabled:opacity-50"
                    />
                  </span>
                  {save.variables?.code === preference.code ? (
                    <span
                      className={`max-w-36 text-right text-xs ${save.isError ? 'text-rose-700' : 'text-slate-500'}`}
                      role={save.isError ? 'alert' : 'status'}
                    >
                      {save.isPending
                        ? t('notifications.saving')
                        : save.isError
                          ? t('notifications.saveFailed')
                          : save.isSuccess
                            ? t('notifications.saved')
                            : null}
                    </span>
                  ) : null}
                </span>
              </label>
            ))}
          </div>
        </section>
      </PageTabPanel>
      <PageTabPanel
        idPrefix="notifications"
        tab="history"
        active={activeTab === 'history'}
      >
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
          <div
            className="mt-4 flex gap-2 md:hidden"
            role="group"
            aria-label={t('notifications.sortBy')}
          >
            {(['openedAt', 'title'] as const).map((field) => (
              <button
                key={field}
                type="button"
                onClick={() => changeSort(field)}
                className={`rounded-lg border px-3 py-2 text-xs font-semibold focus-visible:outline-2 focus-visible:outline-brand-purple ${sort === field ? 'border-brand-purple bg-purple-50 text-brand-purple' : 'border-slate-200 text-slate-600'}`}
              >
                {field === 'openedAt'
                  ? t('notifications.date')
                  : t('notifications.type')}{' '}
                {sort === field ? (order === 'asc' ? '↑' : '↓') : null}
              </button>
            ))}
          </div>
          <ul className="mt-4 divide-y divide-slate-100 md:hidden">
            {list.data?.items.map((item) => (
              <li key={item.id} className="py-4 first:pt-0 last:pb-0">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="flex items-start gap-2 text-sm font-semibold text-slate-900">
                      {!item.readAt ? (
                        <span
                          className="mt-1.5 size-2 shrink-0 rounded-full bg-brand-purple"
                          aria-label={t('notifications.unread')}
                        />
                      ) : null}
                      {item.title}
                    </p>
                    <time
                      dateTime={item.openedAt}
                      className="mt-1 block text-xs text-slate-500"
                    >
                      {new Date(item.openedAt).toLocaleString(i18n.language)}
                    </time>
                  </div>
                  <span
                    className={`shrink-0 rounded-full px-2.5 py-1 text-xs font-semibold ${item.resolvedAt ? 'bg-slate-100 text-slate-600' : 'bg-amber-50 text-amber-900'}`}
                  >
                    {item.resolvedAt
                      ? t('notifications.resolved')
                      : t('notifications.active')}
                  </span>
                </div>
                <p className="mt-3 text-sm leading-6 text-slate-600">
                  {item.description}
                </p>
                <div className="mt-3 flex flex-wrap items-center gap-4 text-sm font-semibold">
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
                      className="text-brand-purple underline"
                      onClick={() => read.mutate(item.id)}
                    >
                      {t('notifications.markRead')}
                    </button>
                  ) : null}
                </div>
              </li>
            ))}
          </ul>
          <div className="mt-4 hidden overflow-x-auto md:block">
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
                    <td className="whitespace-nowrap p-2 text-slate-600">
                      {new Date(item.openedAt).toLocaleString(i18n.language)}
                    </td>
                    <td className="p-2 font-semibold text-slate-900">
                      <span className="flex items-start gap-2">
                        {!item.readAt ? (
                          <span
                            className="mt-1.5 size-2 shrink-0 rounded-full bg-brand-purple"
                            aria-label={t('notifications.unread')}
                          />
                        ) : null}
                        {item.title}
                      </span>
                    </td>
                    <td className="max-w-md p-2 leading-6 text-slate-600">
                      {item.description}
                    </td>
                    <td className="p-2">
                      <span
                        className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${item.resolvedAt ? 'bg-slate-100 text-slate-600' : 'bg-amber-50 text-amber-900'}`}
                      >
                        {item.resolvedAt
                          ? t('notifications.resolved')
                          : t('notifications.active')}
                      </span>
                    </td>
                    <td className="whitespace-nowrap p-2">
                      <div className="flex w-max items-center gap-1">
                        <TableIconLink
                          icon={Eye}
                          label={t('common.view')}
                          to={item.path}
                          onClick={() => {
                            if (!item.readAt) read.mutate(item.id)
                          }}
                        />
                        {!item.readAt ? (
                          <TableIconButton
                            icon={CheckCheck}
                            label={t('notifications.markRead')}
                            onClick={() => read.mutate(item.id)}
                          />
                        ) : null}
                      </div>
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
      </PageTabPanel>
    </div>
  )
}
