import z from "zod";
import { BalanceUpdateSchema, BetUpdateSchema, ChatMessageSchema, MarketUpdateSchema, OnlineUpdateSchema, WsMessage, WsMessageSchema } from "./messages";

export interface WsClientOptions {
    url: string;                     // ws:// or wss:// endpoint
    reconnectInitialDelay?: number;  // ms
    reconnectMaxDelay?: number;      // ms
    reconnectAttempts?: number;      // max tries (0 = infinite)
    debug?: boolean;
}

let client: WSClient | null = null;
export function getWSClient() {
    if (!client) client = new WSClient({ url: process.env.NEXT_PUBLIC_WS_URL! });
    return client;
}


type Handler = (payload: any) => void;
type VoidHandler = () => void;

export class WSClient {
    private url: string;
    private protocols?: string | string[];
    private ws: WebSocket | null = null;

    private handlers = new Map<string, Set<Handler>>();

    private roomSubscribers = new Map<string, Set<string>>();
    private suscriberRooms = new Map<string, Set<string>>();

    private connectHandlers = new Set<VoidHandler>();
    private globalHandlers = new Set<Handler>();

    private reconnectAttempts = 0;
    private reconnectTimer: NodeJS.Timeout | null = null;
    private schemas = new Map<string, z.ZodTypeAny>();
    private options: Required<WsClientOptions>;


    constructor(opts: WsClientOptions) {
        this.url = opts.url;
        this.options = {
            reconnectInitialDelay: opts.reconnectInitialDelay ?? 500,
            reconnectMaxDelay: opts.reconnectMaxDelay ?? 30_000,
            reconnectAttempts: opts.reconnectAttempts ?? 0,
            url: opts.url,
            debug: opts.debug ?? true,
        };

        // Register schemas
        this.registerSchema("balance", BalanceUpdateSchema);
        this.registerSchema("market", MarketUpdateSchema);
        this.registerSchema("bets", BetUpdateSchema);
        this.registerSchema("chat", ChatMessageSchema);
        this.registerSchema("online", OnlineUpdateSchema);

        this.connect();
    }

    private connect() {
        if (this.ws) return;
        this.connectOnce();
    }

    registerSchema<T>(type: string, schema: z.ZodType<T>) {
        this.schemas.set(type, schema);
    }

    private log(...args: any[]) {
        if (this.options.debug)
            console.debug('[WsClient]', ...args);
    }



    private connectOnce() {
        this.log('connecting to', this.url);
        this.ws = new WebSocket(this.url);
        this.ws.onopen = (event) => this.handleOpen(event);
        this.ws.onmessage = (event) => this.handleMessage(event);
        this.ws.onclose = (event) => this.handleClose(event);
        this.ws.onerror = (event) => this.handleError(event);
    }


    private handleOpen(_ev: Event) {
        this.log('open');

        // Call on reconnect handlers
        for (const hand of this.connectHandlers) {
            hand()
        };


        this.reconnectAttempts = 0;
    }

    private handleClose(ev: CloseEvent) {
        this.log('closed', ev.code, ev.reason);
        this.ws = null;
        this.scheduleReconnect();
    }

    private handleError(ev: Event) {
        // browser fires onerror before onclose — we just log
        console.warn('ws error', ev);
    }

    private scheduleReconnect() {
        this.reconnectAttempts += 1;
        const max = this.options.reconnectAttempts;
        if (max > 0 && this.reconnectAttempts > max) {
            this.log('max reconnect attempts reached');
            return;
        }
        const backoff = Math.min(
            this.options.reconnectInitialDelay * Math.pow(1.5, this.reconnectAttempts - 1),
            this.options.reconnectMaxDelay
        );
        this.log('scheduling reconnect in', backoff, 'ms');
        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            this.connectOnce();
        }, backoff);
    }


    private handleMessage(ev: MessageEvent) {
        const text = ev.data as string;
        // support newline-batched messages
        const lines = text.split("\n").map(s => s.trim()).filter(Boolean);
        for (const line of lines) {

            let parsed: unknown;
            try {
                parsed = JSON.parse(line);
            } catch (e) {
                this.log("invalid JSON", line);
                continue;
            }

            const envRes = WsMessageSchema.safeParse(parsed);

            if (!envRes.success) {
                this.log("invalid envelope", envRes.error);
                continue;
            }

            const { type, payload } = envRes.data;

            // Extract type prefix (before the first colon) for schema matching
            const typePrefix = type.split(":")[0];

            const schema = this.schemas.get(typePrefix);
            if (!schema) {
                // no schema registered, ignore
                continue;
            }

            const payloadRes = schema.safeParse(payload);

            if (!payloadRes.success) {
                this.log("payload failed validation", type, payloadRes.error);
                continue;
            }

            const handlersType = this.handlers.get(type);
            if (!handlersType) continue;
            for (const hand of handlersType) hand(payloadRes.data);
        }
    }


    on(type: string, handler: Handler) {
        let set = this.handlers.get(type);
        if (!set) {
            set = new Set();
            this.handlers.set(type, set);
        }
        set.add(handler);
        return () => this.off(type, handler);
    }


    off(type: string, handler: Handler) {
        const set = this.handlers.get(type);
        if (!set) return;
        set.delete(handler);
    }


    onConnect(handler: VoidHandler) {
        this.connectHandlers.add(handler);
        return () => { this.connectHandlers.delete(handler) };
    }

    offConnect(handler: VoidHandler) {
        this.connectHandlers.delete(handler);
    }

    onAny(handler: Handler) {
        this.globalHandlers.add(handler);
        return () => this.globalHandlers.delete(handler);
    }

    emit(msg: WsMessage) {
        const obj = JSON.stringify(msg);
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            try {
                this.ws.send(obj);
                console.log("WebSocket sent", obj);
            } catch (e) {
                console.error("WebSocket send error", e);
            }
        }
    }


}