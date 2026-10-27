import { expect, it, vi } from 'vitest'
import { api } from '../../lib/api'
import { saveUser, userFormSchema } from './api'
it('omits an unchanged password when updating an internal user', async () => {
  const put = vi.spyOn(api, 'put').mockResolvedValue({ data: {} })
  await saveUser(42, {
    name: 'Ana',
    email: 'ana@example.test',
    role: 'viewer',
    active: true,
    password: '',
  })
  expect(put).toHaveBeenCalledWith('/admin/users/42', {
    name: 'Ana',
    email: 'ana@example.test',
    role: 'viewer',
    active: true,
    password: undefined,
  })
  put.mockRestore()
})
it('rejects a weak optional replacement password', () => {
  expect(
    userFormSchema.safeParse({
      name: 'Ana',
      email: 'ana@example.test',
      role: 'viewer',
      active: true,
      password: 'short',
    }).success,
  ).toBe(false)
})

it('requires 15 characters for new system passwords and allows unchanged passwords', () => {
  const body = {
    name: 'Ana',
    email: 'ana@example.test',
    role: 'admin',
    active: true,
  }
  expect(
    userFormSchema.safeParse({ ...body, password: 'abcdefghijklm' }).success,
  ).toBe(false)
  expect(
    userFormSchema.safeParse({ ...body, password: 'uma frase longa de acesso' })
      .success,
  ).toBe(true)
  expect(userFormSchema.safeParse({ ...body, password: '' }).success).toBe(true)
})

it('does not count surrounding spaces toward the password minimum', () => {
  expect(
    userFormSchema.safeParse({
      name: 'Ana',
      email: 'ana@example.test',
      role: 'admin',
      active: true,
      password: 'ab             ',
    }).success,
  ).toBe(false)
})

it('preserves the original password when validating and submitting', async () => {
  const password = '  uma frase longa de acesso  '
  const body = {
    name: 'Ana',
    email: 'ana@example.test',
    role: 'admin' as const,
    active: true,
    password,
  }
  expect(userFormSchema.parse(body).password).toBe(password)
  const post = vi.spyOn(api, 'post').mockResolvedValue({ data: {} })
  try {
    await saveUser(undefined, body)
    expect(post).toHaveBeenCalledWith('/admin/users', body)
  } finally {
    post.mockRestore()
  }
})
