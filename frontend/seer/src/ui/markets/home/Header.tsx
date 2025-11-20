"use client";

import { Category, sortOptions, statusOptions } from "@/lib/definitions";
import { useUpdateSearchParams } from "@/lib/hooks/useUpdateSearchParams";
import MenuVertical from "@/ui/menu_small_vertical/MenuVertical";
import { ArrowDownNarrowWide, Clock } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";



export function Header({ activeCategory }: { activeCategory: Category }) {

    const params = useSearchParams();
    const sort = params.get("sort") ?? "trending";
    const status = params.get("status") ?? "active";
    const router = useRouter();

    const { setParams } = useUpdateSearchParams();

    return (
        <div className="mb-4">

            <div className="flex items-center gap-5">
                {/* <div className="flex items-center gap-2">
                    <Image src={activeCategory.iconUrl} alt={activeCategory.label} width={26} height={26} className="filter-(--primary-blue-filter) w-5.5 h-5.5" />
                    <Heading>{activeCategory.label}</Heading>
                </div> */}

                <div className="flex items-center gap-2.5 md:gap-3">

                    <div className="md:w-38">
                        <MenuVertical leftPart={<Clock className="w-4 h-4 text-gray-500 mt-0.5" strokeWidth={2.5} />}
                            choices={statusOptions}
                            value={status}
                            onChange={(value) => setParams({ status: value })}
                            height="h-11"
                            makeResponsive
                            widthResponsive="w-35"
                            positionResponsive="-left-24.5" />
                    </div>

                    <div className="md:w-40">
                        <MenuVertical leftPart={<ArrowDownNarrowWide className="w-4 h-4 text-gray-500 mt-0.5" strokeWidth={2.5} />}
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