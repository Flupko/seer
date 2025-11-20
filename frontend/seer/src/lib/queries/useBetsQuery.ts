import { useInfiniteQuery } from "@tanstack/react-query";
import { getUserBets } from "../api";
import { UserBetSearch, UserBetsRes } from "../definitions";
import { getNextPageParamFromMetadata } from "../meta";
import { betSearchKey } from "./betSearchKey";


export const useBetsQuery = ({ search }: { search: UserBetSearch }) => {
    return useInfiniteQuery({
        queryKey: betSearchKey(search),
        queryFn: ({ pageParam = 1 }) => getUserBets({ ...search, page: pageParam } as UserBetSearch),
        getNextPageParam: (lastPage: UserBetsRes) => getNextPageParamFromMetadata(lastPage.metadata),
        initialPageParam: 1,
        staleTime: 5 * 60 * 1000,
    });
}