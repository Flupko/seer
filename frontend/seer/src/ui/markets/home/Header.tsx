"use client";

import { Category, sortOptions, statusOptions } from "@/lib/definitions";
import { useUpdateSearchParams } from "@/lib/hooks/useUpdateSearchParams";
import MenuVertical from "@/ui/menu_small_vertical/MenuVertical";
import { ArrowDownNarrowWide, Clock } from "lucide-react";
import Image from "next/image";
import { useRouter, useSearchParams } from "next/navigation";



export function Header({ activeCategory }: { activeCategory: Category }) {

    const params = useSearchParams();
    const sort = params.get("sort") ?? "trending";
    const status = params.get("status") ?? "active";
    const router = useRouter();

    const { setParams } = useUpdateSearchParams();

    const updateQuery = (key: string, value: string) => {
        const newParams = new URLSearchParams(params.toString());
        newParams.set(key, value);
        const queryString = newParams.toString();
        const url = queryString ? `/?${queryString}` : '/';
        router.push(url);
    }


    return (
        <div className="mb-4">

            <div className="flex items justify-between">
                <div className="flex items-center gap-2">
                    <Image src={activeCategory.iconUrl} alt={activeCategory.label} width={28} height={28} className="filter-(--primary-blue-filter)" />
                    <h3 className="text-[1.375rem] font-bold text-white">{activeCategory.label}</h3>
                </div>

                <div className="flex items-center gap-3">

                    <div className="w-50">
                        <MenuVertical leftPart={<Clock className="w-5 h-5" strokeWidth={1.7} />}
                            choices={statusOptions}
                            value={status}
                            onChange={(value) => setParams({ status: value })}
                            height="h-11" />
                    </div>

                    <div className="w-50">
                        <MenuVertical leftPart={<ArrowDownNarrowWide className="w-5 h-5" strokeWidth={1.7} />}
                            choices={sortOptions}
                            value={sort}
                            onChange={(value) => setParams({ sort: value })}
                            height="h-11" />
                    </div>

                </div>

            </div>


        </div>
    )
}