import { ChatMessage } from "@/socket/messages";
import { create } from "zustand/react";


const MAX_MESSAGES_PER_CHAT = 20;

// Chats with parity tracking to allow for effect where binary color changing for each new message
interface ChatStore {
    chats: Record<string, { "messages": ChatMessage[], "parity": number }>;
    addChatMessage: (message: ChatMessage) => void;
}

export const useChatStore = create<ChatStore>((set, get) => ({
    chats: {},
    addChatMessage: (message: ChatMessage) => {
        set((state) => {

            const slug = message.chatSlug;
            if (!state.chats[slug]) {
                state.chats[slug] = {
                    messages: [],
                    parity: 0,
                }
            }
            // Update parity
            state.chats[slug].parity = (state.chats[slug].parity + 1) % 2;

            // Add message
            const chatMessages = state.chats[slug].messages;
            chatMessages.unshift(message);

            if (chatMessages.length > MAX_MESSAGES_PER_CHAT) {
                chatMessages.pop();
            }

            return {
                chats: {
                    ...state.chats,
                },
            };


        });
    },
}));
