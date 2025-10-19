"use client"

import { APIError } from "@/lib/api";
import { toastStyled } from "@/ui/Toast";
import { isServer, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";

function makeQueryClient() {
    return new QueryClient({
        queryCache: new QueryCache({
            onError: (error: Error | APIError, query) => {
                console.error(`Error on query ${query.queryKey}:`, error);
                toastStyled("Failed to fetch", { type: "error" });
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
}

let browserQueryClient: QueryClient | undefined = undefined

function getQueryClient() {
    if (isServer) {
        // Server: always make a new query client
        return makeQueryClient()
    } else {
        // Browser: make a new query client if we don't already have one
        // This is very important, so we don't re-make a new client if React
        // suspends during the initial render. This may not be needed if we
        // have a suspense boundary BELOW the creation of the query client
        if (!browserQueryClient) browserQueryClient = makeQueryClient()
        return browserQueryClient
    }
}

export default function Providers({ children }: { children: React.ReactNode }) {
    // NOTE: Avoid useState when initializing the query client if you don't
    //       have a suspense boundary between this and the code that may
    //       suspend because React will throw away the client on the initial
    //       render if it suspends and there is no boundary
    const queryClient = getQueryClient()

    return (
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )
}