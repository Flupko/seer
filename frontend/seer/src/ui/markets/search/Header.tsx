"use client";

import { sortOptions, statusOptions } from "@/lib/definitions";
import { useUpdateSearchParams } from "@/lib/hooks/useUpdateSearchParams";
import MenuVertical from "@/ui/menu_small_vertical/MenuVertical";
import { ArrowDownNarrowWide, Clock } from "lucide-react";
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

            <div className="flex items-center justify-between">
                <div className="flex items-baseline gap-4 mt-1.5">
                    <span className="text-2xl font-bold">{query}</span>
                    <span className="text-md text-gray-400">{nbResults} result{nbResults !== 1 ? 's' : ''}</span>
                </div>

                <div className="flex items-center gap-2.5 md:gap-3">

                    <div className="md:w-50">
                        <MenuVertical leftPart={<Clock className="w-5 h-5" strokeWidth={1.7} />}
                            choices={statusOptions}
                            value={status}
                            onChange={(value) => setParams({ status: value })}
                            height="h-11"
                            makeResponsive
                            widthResponsive="w-35"
                            positionResponsive="-left-24.5" />
                    </div>

                    <div className="md:w-50">
                        <MenuVertical leftPart={<ArrowDownNarrowWide className="w-5 h-5" strokeWidth={1.7} />}
                            choices={sortOptions}
                            value={sort}
                            onChange={(value) => setParams({ sort: value })}
                            height="h-11"
                            makeResponsive
                            widthResponsive="w-40"
                            positionResponsive="-left-29.5" />
                    </div>

                </div>

            </div>


        </div>
    )
}