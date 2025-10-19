'use client'

import { QueryFunction, QueryKey, useQueryClient } from '@tanstack/react-query'
import Link from 'next/link'
import { useRef } from 'react'

type PrefetchItem<TData = unknown> = {
    queryKey: QueryKey
    queryFn: QueryFunction<TData>
    staleTime?: number
}

export default function PrefetchLink({
    href,
    queries = [],
    children,
}: {
    href: string
    queries?: PrefetchItem[]
    children: React.ReactNode
}) {
    const queryClient = useQueryClient()
    const alreadyFetched = useRef(false)

    const onIntent = () => {
        if (alreadyFetched.current) return

        const tasks = queries
            .map((q) =>
                queryClient.prefetchQuery({
                    queryKey: q.queryKey,
                    queryFn: q.queryFn,
                    staleTime: q.staleTime ?? 60_000,
                }),
            )

        void Promise.all(tasks)
    }

    return (
        <Link href={href} onMouseEnter={onIntent} onFocus={onIntent} onTouchStart={onIntent} className='contents'>
            {children}
        </Link>
    )
}
