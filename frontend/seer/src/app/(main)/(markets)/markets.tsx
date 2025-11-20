"use client";

import { MarketSearchRes, searchMarket } from "@/lib/api";
import { Category, MarketSearch } from "@/lib/definitions";
import { getNextPageParamFromMetadata } from "@/lib/meta";
import { marketSearchKey } from "@/lib/queries/marketSearchKey";
import { Header } from "@/ui/markets/home/Header";
import MarketsDisplay from "@/ui/markets/home/MarketsDisplay";
import { useInfiniteQuery } from "@tanstack/react-query";

export default function MarketsHome({ search, activeCategory }: { search: MarketSearch, activeCategory: Category }) {

    const {
        data,
        isLoading,
        isError,
        fetchNextPage,
        isFetchingNextPage,
        hasNextPage,
    } = useInfiniteQuery({
        queryKey: marketSearchKey(search),
        queryFn: ({ pageParam = 1 }) => searchMarket({ ...search, page: pageParam } as MarketSearch),
        getNextPageParam: (lastPage: MarketSearchRes) => getNextPageParamFromMetadata(lastPage.metadata),
        initialPageParam: 1,
        staleTime: 5 * 60 * 1000,
    });

    // const nbResults = data?.pages[-1].metadata.totalRecords || 0

    const markets = data?.pages.flatMap((p) => p.markets)

    console.log(markets);
    return (
        <div className="pt-3 lg:pt-6 transition-all">
            <Header activeCategory={activeCategory} />
            <MarketsDisplay markets={markets} {...{ isLoading, fetchNextPage, isFetchingNextPage, hasNextPage }} />
        </div>
    )
}
