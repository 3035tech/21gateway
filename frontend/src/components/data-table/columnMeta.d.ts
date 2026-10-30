import type { RowData } from '@tanstack/react-table'

declare module '@tanstack/react-table' {
  // TanStack requires both generic parameters for declaration merging.
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  interface ColumnMeta<TData extends RowData, TValue> {
    compact?: boolean
  }
}
