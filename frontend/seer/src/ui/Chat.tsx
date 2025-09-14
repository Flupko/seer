"use client"

import { useEffect, useState } from "react"
import {socket} from "@/socket/socket"

export default function Chat() {

    const [messageInput, setMessageInput] = useState<string>("")
    const [messages, setMessages] = useState<string[]>([])

    useEffect(() => {
        socket.onmessage = (message) => {
            setMessages(prev => [...prev, message.data])
        }
    }, [])

    const sendMessage = () => {
        if (socket.readyState === WebSocket.OPEN) {
            socket.send(messageInput)
            setMessageInput("")
        } else {
            console.error("WebSocket is not open.")
        }
    }

    return (
        <div>
            <h1>WebSocket Chat</h1>
            <div>
                <input
                    type="text"
                    value={messageInput}
                    onChange={(e) => setMessageInput(e.target.value)}
                />
                <button className="p-3 bg-amber-200" onClick={sendMessage}>Send</button>
            </div>
            <div>
                <h2>Messages:</h2>
                <ul>
                    {messages.map((msg, index) => (
                        <li key={index}>{msg}</li>
                    ))}
                </ul>
            </div>
        </div>
    )

}
