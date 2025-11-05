import { useWebSocket } from "@/app/WsProvider";
import { useUpdateSearchParams } from "@/lib/hooks/useUpdateSearchParams";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useChatStore } from "@/lib/stores/chats";
import { useModalStore } from "@/lib/stores/modal";
import { useOnlineStore } from "@/lib/stores/online";
import { SendHorizontal, X } from "lucide-react";
import { motion } from "motion/react";
import { useEffect, useState } from "react";
import DrawerFooter from "../drawer/DrawerFooter";
import DrawerHeader from "../drawer/DrawerHeader";
import Input from "../Input";
import ProfilePicture from "../ProfilePicture";
import { toastStyled } from "../Toast";

export default function ChatDrawer() {


    const [inputValue, setInputValue] = useState("");

    const sendEnabled = inputValue.trim().length > 0 && inputValue.length <= 50;

    const handleSend = () => {
        if (!sendEnabled) return;
        ws?.emit({ type: "send:chat", payload: { message: inputValue, chatSlug: "global" } });
        setInputValue("");
    }


    const openModal = useModalStore((state) => state.openModal)

    const user = useUserQuery().data;

    const { setParams } = useUpdateSearchParams();


    const ws = useWebSocket();
    // On mount add the listener
    useEffect(() => {
        if (!ws) return;

        console.log("ChatDrawer mounting, subscribing to rate limited chat messages");

        const off = ws.on("rate:chat:global", () => {
            console.log("rate limited chat message");
            toastStyled("You're sending messages too quickly. Please slow down.", { type: "error" });
        });

        return () => { off(); }

    }, [ws]);



    const onlineCount = useOnlineStore((state) => state.onlineCount);

    const handleShowUserprofile = (username: string) => {
        setParams({ show: 'user', username });
    }


    const chats = useChatStore((state) => state.chats)
    const globalChatMessages = chats?.global?.messages || []
    const globalParity = chats?.global?.parity || 0

    return (
        <div className="flex flex-col h-full relative">
            <DrawerHeader title="Chat" />

            {/* Outcomes select input */}

            <div className="flex flex-col-reverse overflow-y-auto grow scrollbar-light">
                {
                    globalChatMessages.map((m, idx) =>
                        <div key={m.id} className={`p-3 flex items-start gap-2 text-sm ${idx % 2 == globalParity ? "bg-gray-800/10 hover:bg-gray-800/20" : "bg-gray-800/40 hover:bg-gray-800/50"} `}>
                            <div className="cursor-pointer" onClick={() => handleShowUserprofile(m.user.username)}>
                                <ProfilePicture url={m.user.profileImageKey} size={30} />
                            </div>

                            <div className="flex flex-col gap-2">
                                <span className="font-bold hover:text-primary-blue cursor-pointer transition-colors" onClick={() => handleShowUserprofile(m.user.username)}>{m.user.username}</span>
                                <span className="text-gray-300 font-normal">{m.content}</span>
                            </div>
                        </div>)
                }
            </div>


            <div className="grow shrink-0 relative mt-auto">


                <DrawerFooter>

                    <div className="flex flex-col gap-3">
                        <Input
                            placeholder="Your message here"
                            disabled={!user}
                            value={inputValue}
                            onChange={(e) => setInputValue(e.target.value)}
                            onKeyDown={(e) => {
                                if (e.key === "Enter") {
                                    handleSend();
                                }
                            }}
                            rightEl={<SendHorizontal

                                onClick={handleSend}
                                className={`w-5 h-5 ${sendEnabled ? "text-gray-400 cursor-pointer hover:text-gray-500" : "text-gray-600"} transition-colors`}
                                strokeWidth={1.5} />} />

                        <span className="text-sm text-gray-400 flex items-center gap-2">
                            <span className="rounded-full bg-green-800 w-1.5 h-1.5"></span> {onlineCount} Online
                        </span>
                    </div>
                </DrawerFooter>
            </div>



        </div >

    )




}

function ErrorDrawer({ children }: { children: React.ReactNode }) {
    return (
        <motion.div className="text-sm font-bold bg-error py-3.5 px-5 rounded-lg flex items-center gap-3"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2, ease: "easeIn" }} >

            <span className="bg-white rounded-full p-[0.2rem]">
                <X className="w-2.5 h-2.5 text-error" strokeWidth={5} />
            </span>
            {children}
        </motion.div>
    )
}