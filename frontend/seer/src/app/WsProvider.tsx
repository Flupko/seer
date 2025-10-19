"use client";

import { getWSClient, WSClient } from "@/socket/socket";
import { createContext, useContext, useEffect, useState } from "react";


interface WsContext {
    wsClient: WSClient | null;
}


const WsContext = createContext<WsContext>({
    wsClient: null,
});

// Create custom hook
export function useWebSocket() {
    const context = useContext(WsContext);
    return context.wsClient;
}

export function WsProvider({ children }: { children: React.ReactNode }) {

    const [wsClient, setWsClient] = useState<WSClient | null>(null);

    useEffect(() => {
        const ws = getWSClient() // only runs in browser

        console.log("WsProvider mounting, subscribing to ws events");
        ws.onConnect(() => {
            ws.emit({ type: "subscribe:markets" });
            ws.emit({ type: "subscribe:online_count" });
            ws.emit({ type: "subscribe:bets" });
        });

        setWsClient(ws)
    }, [])

    return (
        <WsContext.Provider value={{ wsClient: wsClient }}>
            {children}
        </WsContext.Provider>
    )
}