'use client'

import { QueryFunction, QueryKey, useQueryClient } from '@tanstack/react-query'
import Link, { LinkProps } from 'next/link'
import { useRef } from 'react'

type PrefetchItem<TData = unknown> = {
    queryKey: QueryKey
    queryFn: QueryFunction<TData>
    staleTime?: number
}

type PrefetchLinkProps = LinkProps & {
    queries?: PrefetchItem[]
    infiniteQueries?: PrefetchItem[]
    children: React.ReactNode
    className?: string
}

export default function PrefetchLink({
    href,
    queries = [],
    infiniteQueries = [],
    children,
    className,
    ...rest
}: PrefetchLinkProps) {
    const queryClient = useQueryClient()
    const prefetched = useRef<boolean>(false)


    const onIntent = async () => {
        if (prefetched.current) return
        prefetched.current = true

        try {
            const tasks = [
                ...queries.map((q) =>
                    queryClient.prefetchQuery({
                        queryKey: q.queryKey,
                        queryFn: q.queryFn,
                        staleTime: q.staleTime ?? 60_000,
                    }),
                ),
                ...infiniteQueries.map((q) =>
                    queryClient.prefetchInfiniteQuery({
                        queryKey: q.queryKey,
                        initialPageParam: 1 as never,
                        queryFn: q.queryFn,
                    }),
                ),
            ]
            await Promise.all(tasks)
        } catch (e) {
            // optional: log or ignore prefetch errors
            console.warn('Prefetch failed for', href, e)
        }
    }

    return (
        <Link
            href={href}
            onMouseEnter={onIntent}
            onPointerEnter={onIntent}
            onFocus={onIntent}
            onTouchStart={onIntent}
            className={className}
            {...rest}
        >
            {children}
        </Link>
    )
}
