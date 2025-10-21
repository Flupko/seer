'use client';

import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useCallback } from 'react';

type UpdateValue = string | number | boolean | null | undefined;
type Updates = Record<string, UpdateValue>;
type Options = { replace?: boolean; scroll?: boolean };

export function useUpdateSearchParams() {
    const router = useRouter();
    const pathname = usePathname();
    const searchParams = useSearchParams();
    // merge updates into current params and navigate without scrolling
    const setParams = useCallback(
        (updates: Updates, opts: Options = {}) => {
            const params = new URLSearchParams(searchParams); // copy current params
            for (const [k, v] of Object.entries(updates)) {
                if (v === undefined || v === null || v === '') params.delete(k);
                else params.set(k, String(v));
            }
            const qs = params.toString();
            const href = qs ? `${pathname}?${qs}` : pathname;
            const scroll = opts.scroll ?? false;
            if (opts.replace ?? true) router.replace(href, { scroll });
            else router.push(href, { scroll });
        },
        [router, pathname, searchParams]
    );

    return { setParams };
}
