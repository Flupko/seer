import { useQuery } from "@tanstack/react-query";
import { getBalance } from "../api";
import { Currency } from "../definitions";
import { useUserQuery } from "./useUserQuery";


export const useBalanceQuery = (currency: Currency) => {
    const { data: user } = useUserQuery();
    return useQuery({
        queryKey: ["balance", currency],
        queryFn: () => getBalance(currency),
        retry: false,
        staleTime: 60, // 1 minute
        enabled: !!user, // only fetch if user is logged in
    });
}