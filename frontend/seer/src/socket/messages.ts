import { DecimalSchema } from "@/lib/definitions";
import z from "zod";

export const WsMessageSchema = z.object({
    type: z.string().max(100),
    payload: z.any().optional(),
});

export type WsMessage = z.infer<typeof WsMessageSchema>;

export const BalanceUpdateSchema = z.object({
    currency: z.string(),
    balance: DecimalSchema,
    version: z.number().int(),
});
export type BalanceUpdate = z.infer<typeof BalanceUpdateSchema>;

export const OutcomeUpdateSchema = z.object({
    id: z.number().int(),
    quantity: DecimalSchema,
});
export type OutcomeUpdate = z.infer<typeof OutcomeUpdateSchema>;

export const MarketUpdateSchema = z.object({
    marketID: z.uuid(),
    marketVersion: z.number().int(),
    outcomes: z.array(OutcomeUpdateSchema),
});
export type MarketUpdate = z.infer<typeof MarketUpdateSchema>;

export const WsUserSchema = z.object({
    id: z.uuid(),
    username: z.string(),
    profileImageKey: z.string().optional(),
});

export const ChatMessageSchema = z.object({
    id: z.uuid(),
    chatSlug: z.string(),
    content: z.string(),
    createdAt: z.coerce.date(),
    type: z.enum(['user', 'system']),
    user: WsUserSchema,
});


export const BetUpdateSchema = z.object({
    id: z.uuid(),
    marketID: z.uuid(),
    marketName: z.string(),
    outcomeId: z.number(),
    outcomeName: z.string(),
    wager: DecimalSchema,
    payout: DecimalSchema,
    avgPrice: DecimalSchema,
    placedAt: z.coerce.date(),
    user: WsUserSchema.optional(),
});

export const OnlineUpdateSchema = z.object({
    usersOnlineCount: z.number().int(),
});
export type OnlineUpdate = z.infer<typeof OnlineUpdateSchema>;

export const WSErrorSchema = z.object({
    error: z.string(),
});
export type WSError = z.infer<typeof WSErrorSchema>;

export const ChatRoomPrefix = 'chat:';
export const MarketRoomPrefix = 'market:';

export const getMarketRoom = (marketId: string) => `${MarketRoomPrefix}${marketId}`;
export const getChatRoom = (chatSlug: string) => `${ChatRoomPrefix}${chatSlug}`;
