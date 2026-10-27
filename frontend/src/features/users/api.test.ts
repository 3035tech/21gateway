import { expect, it, vi } from 'vitest'
import { api } from '../../lib/api'
import { saveUser, userFormSchema } from './api'

const body = {
  name: 'Ana',
  email: 'ana@example.test',
  role: 'admin' as const,
  active: true,
}

it('creates an internal user by invitation without collecting a password', async () => {
  expect(userFormSchema.parse(body)).toEqual(body)
  const post = vi.spyOn(api, 'post').mockResolvedValue({ data: {} })
  try {
    await saveUser(undefined, body)
    expect(post).toHaveBeenCalledWith('/admin/users/invite', body)
  } finally {
    post.mockRestore()
  }
})

it('updates user details without submitting a password', async () => {
  const put = vi.spyOn(api, 'put').mockResolvedValue({ data: {} })
  try {
    await saveUser(42, body)
    expect(put).toHaveBeenCalledWith('/admin/users/42', body)
  } finally {
    put.mockRestore()
  }
})
