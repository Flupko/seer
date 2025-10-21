import { getUser } from "@/lib/api";
import { useQuery } from "@tanstack/react-query";

export const useUserQuery = () => {
    return useQuery({
        queryKey: ["user"],
        queryFn: () => getUser(),
        retry: false,
        staleTime: 60 * 60 * 1000, // 1 hour
    });
}