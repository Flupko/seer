import { API_BASE_URL } from "@/lib/api";
import { useState } from "react";
import Google from "../../../public/google.svg";
import Twitch from "../../../public/twitch.svg";
import Button from "../Button";
import { toastStyled } from "../Toast";

export default function Providers() {

    const [loadingProvider, setLoadingProvider] = useState<'google' | 'twitch' | null>(null);
    const handleClick = async (provider: 'google' | 'twitch') => {
        setLoadingProvider(provider);
        try {
            const response = await fetch(`${API_BASE_URL}/auth/provider/${provider}`, { credentials: "include" });
            if (!response.ok) {
                setLoadingProvider(null);
                toastStyled("Failed to authenticate", { type: "error" });
                return;
            }
            const { url } = await response.json();
            // Redirect to OAuth provider
            window.location.href = url;
        } catch (error) {
            setLoadingProvider(null);
            toastStyled("Failed to authenticate", { type: "error" });
        }

    }
    return (
        <div className="space-y-6">

            {/* Horizontal line with text in middle */}
            <div className="flex items-center w-full gap-3">
                <div className="flex-grow h-[1px] bg-gray-600"></div>
                <span className="text-white text-sm">Or continue with</span>
                <div className="flex-grow h-[1px] bg-gray-600"></div>
            </div>

            <div className="flex gap-3">
                <Button bg="bg-transparent" width="full" height="large" onClick={() => handleClick('google')} className="border border-gray-600 flex-1 hover:bg-gray-800" isLoading={loadingProvider === 'google'}>
                    <div className="flex items-center justify-center gap-3">
                        <Google className="w-5 h-5" />
                        Google
                    </div>

                </Button>

                <Button bg="bg-transparent" width="full" height="large" onClick={() => handleClick('twitch')} className="border border-gray-600 flex-1 hover:bg-gray-800" isLoading={loadingProvider === 'twitch'}>
                    <div className="flex items-center justify-center gap-3">
                        <Twitch className="w-5 h-5 brightness-105" />
                        Twitch
                    </div>
                </Button>
            </div>
        </div>
    )
}