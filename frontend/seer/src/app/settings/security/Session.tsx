import { revokeSession } from "@/lib/api";
import { Session } from "@/lib/definitions";
import { timeSince } from "@/lib/utils/date";
import { toastStyled } from "@/ui/Toast";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export default function SessionDisplay({ session }: { session: Session }) {

    const queryClient = useQueryClient();

    const { mutate } = useMutation({
        mutationFn: revokeSession,
        onSuccess: () => {
            // Purge the session from the list by refetching
            queryClient.invalidateQueries({ queryKey: ["user", "sessions"] });
        },
        onError: () => {
            toastStyled("Error revoking session", { type: "error" });
        },
    });

    return (
        <tr className={`rounded-xl text-white transition-all`}>
            <SessionEl current={session.current}>{(session.browser && session.os) ? `${session.browser} (${session.os})` : "-"}</SessionEl>
            <SessionEl current={session.current}>{(session.city && session.country) ? `${session.city} (${session.country})` : "-"}</SessionEl>
            <SessionEl current={session.current}>{session.ip ?? "-"}</SessionEl>
            <SessionEl current={session.current}>{session.current ? "Online" : timeSince(session.lastUsedAt)}</SessionEl>
            <SessionEl current={session.current}>
                {session.current ? "Current" : session.active ?
                    <span className="text-red-400 cursor-pointer hover:text-red-300 transition-colors" onClick={() => mutate(session.id)}>Revoke Session</span>
                    : <span className="text-gray-400">Logged out</span>}
            </SessionEl>
        </tr>
    )
}

function SessionEl({ children, current }: { children: React.ReactNode, current: boolean }) {
    return (
        <td className={`text-white text-left font-bold text-sm first:rounded-l-lg last:rounded-r-lg ${current ? "bg-gray-800" : "bg-grayscale-black"} px-6 py-5`}>
            {children}
        </td>
    )
}

