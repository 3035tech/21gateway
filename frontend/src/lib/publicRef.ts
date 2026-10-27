import { z } from 'zod'

export function isPublicRef(value: unknown): value is string {
  return z.uuid().safeParse(value).success
}
