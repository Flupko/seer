"use client";

import { sortOptions, statusOptions } from "@/lib/definitions";
import { useUpdateSearchParams } from "@/lib/hooks/useUpdateSearchParams";
import MenuVertical from "@/ui/menu_small_vertical/MenuVertical";
import { useRouter, useSearchParams } from "next/navigation";



export function Header({ nbResults }: { nbResults: number }) {

    const params = useSearchParams();
    const query = params.get("q")
    const sort = params.get("sort") ?? "trending";
    const status = params.get("status") ?? "active";
    const router = useRouter();

    const { setParams } = useUpdateSearchParams();

    return (
        <div className="mb-4">

            <div className="flex flex-col gap-5">
                <div className="flex items-baseline gap-4">
                    <span className="text-2xl font-bold">{query}</span>
                    <span className="text-md text-gray-400">{nbResults} result{nbResults !== 1 ? 's' : ''}</span>
                </div>

                <div className="flex items-center gap-3 overflow-x-auto">

                    <div className="shrink-0">
                        <MenuVertical leftPart={<span className="text-gray-400 mr-1.5">Status:</span>}
                            choices={statusOptions.map(({ value, element }) => ({ value, element: <span className="text-sm font-medium">{element}</span> }))}
                            value={status}
                            menuWidth={170}
                            onChange={(value) => setParams({ status: value })}
                            height="h-10"
                            padding="px-3" />
                    </div>

                    <div className="shrink-0">
                        <MenuVertical leftPart={<span className="text-gray-400 mr-1.5">Sort:</span>}
                            choices={sortOptions.map(({ value, element }) => ({ value, element: <span className="text-sm font-medium">{element}</span> }))}
                            value={sort}
                            menuWidth={180}
                            onChange={(value) => setParams({ sort: value })}
                            height="h-10"
                            padding="px-3" />
                    </div>

                </div>

            </div>


        </div>
    )
}