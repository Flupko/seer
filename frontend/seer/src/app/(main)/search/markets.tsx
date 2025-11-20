"use client";

import { MarketSearchRes, searchMarket } from "@/lib/api";
import { MarketSearch } from "@/lib/definitions";
import { getNextPageParamFromMetadata } from "@/lib/meta";
import { marketSearchKey } from "@/lib/queries/marketSearchKey";
import MarketsDisplay from "@/ui/markets/home/MarketsDisplay";
import { Header } from "@/ui/markets/search/Header";
import { useInfiniteQuery } from "@tanstack/react-query";

export default function MarketsSearch({ search }: { search: MarketSearch }) {

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

    const nbResults = data?.pages[0].metadata.totalRecords || 0
    const markets = data?.pages.flatMap((p) => p.markets)

    console.log(markets)

    return (
        <div className="transition-all">
            <Header nbResults={nbResults} />
            <MarketsDisplay markets={markets} {...{ isLoading, fetchNextPage, isFetchingNextPage, hasNextPage }} />
        </div>
    )
}
