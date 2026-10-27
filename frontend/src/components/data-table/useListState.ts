import { useState } from 'react'
import type {
  OnChangeFn,
  PaginationState,
  SortingState,
} from '@tanstack/react-table'
import { DEFAULT_PAGE_SIZE } from './constants'

export function useListState(initialSorting: SortingState = []) {
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: DEFAULT_PAGE_SIZE,
  })
  const [sorting, setSorting] = useState<SortingState>(initialSorting)
  const [search, setSearch] = useState('')
  const onSortingChange: OnChangeFn<SortingState> = (updater) => {
    setSorting(updater)
    setPagination((current) => ({ ...current, pageIndex: 0 }))
  }
  const onSearchChange = (value: string) => {
    setSearch(value)
    setPagination((current) => ({ ...current, pageIndex: 0 }))
  }
  return {
    pagination,
    onPaginationChange: setPagination,
    sorting,
    onSortingChange,
    search,
    onSearchChange,
  }
}
