import { expect, it, vi } from 'vitest'
import { AxiosError } from 'axios'
import { api } from '../../lib/api'
import { getSession } from './api'
it('treats only a 401 as anonymous and propagates infrastructure failures', async () => {
  const spy = vi.spyOn(api, 'get')
  spy.mockRejectedValueOnce(
    new AxiosError('private cause', undefined, undefined, undefined, {
      status: 401,
    } as never),
  )
  expect(await getSession()).toBeNull()
  spy.mockRejectedValueOnce(
    new AxiosError('private cause', undefined, undefined, undefined, {
      status: 503,
    } as never),
  )
  await expect(getSession()).rejects.toThrow()
  spy.mockRestore()
})
