"use client"

import { QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "react-toastify";
import { ApiError } from "next/dist/server/api-utils";
import { APIError } from "@/lib/api";
import { toastStyled } from "@/ui/Toast";

interface Props {
    children: React.ReactNode;
}

export default function ReactQueryProvider({ children }: Props) {
    const [queryClient] = useState(() =>
        new QueryClient({
            queryCache: new QueryCache({
                onError: (error: Error | APIError, query) => {
                    toastStyled("Failed to fetch", { type: "error"});
                },
            }),
            defaultOptions: {
                queries: {
                    retryDelay: (attempt) => Math.min(500 * 2 ** attempt, 30000),
                    retry: false,
                    refetchOnWindowFocus: false,
                },
            },
        })
    );

    return (
        <QueryClientProvider client={queryClient}>
            {children}
        </QueryClientProvider>);
}