"use client"

import { getSessions, revokeSession } from "@/lib/api";
import { Session } from "@/lib/definitions";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useModalStore } from "@/lib/stores/modal";
import { timeSince } from "@/lib/utils/date";
import ContainerSmall from "@/ui/containers/ContainerSmall";
import MenuInput from "@/ui/menu_small_vertical/MenuVertical";
import { TableCell, TableHead, TableHeading, TableRow } from "@/ui/Table";
import { toastStyled } from "@/ui/Toast";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronRight } from "lucide-react";
import { useState } from "react";
import Google from '../../../../public/google.svg';
import Lock from "../../../../public/lock.svg";
import Twitch from '../../../../public/twitch.svg';
import Container from "../Container";


export default function SecurityPage() {

    const openModal = useModalStore((state) => state.openModal)

    const [showInactive, setShowInactive] = useState(false);

    const { data, isPending } = useQuery({
        queryKey: ["user", "sessions", "showInactive", showInactive],
        queryFn: () => getSessions(showInactive),
        staleTime: Infinity,
    })

    const { data: user } = useUserQuery()


    return (
        <div className="flex flex-col gap-5">


            <div className="flex flex-col gap-5 lg:flex-row">

                {user?.providerId !== "credentials" &&
                    <div className="flex-1">

                        <Container title="Linked Account">
                            <div className="select-none">
                                <ContainerSmall>
                                    <div className="flex justify-between items-center">
                                        <div className="flex gap-3 items-center">
                                            {user?.providerId === "google" && <Google className="w-4.5 h-4.5" />}
                                            {user?.providerId === "twitch" && <Twitch className="w-4.5 h-4.5" />}
                                            <span className="font-medium font-lg">
                                                {user?.providerId
                                                    ? user.providerId.charAt(0).toUpperCase() + user.providerId.slice(1)
                                                    : ""}
                                            </span>
                                        </div>
                                        <span className="text-gray-300 font-medium text-sm">Linked</span>

                                    </div>
                                </ContainerSmall>
                            </div>

                        </Container>
                    </div>

                }

                <div className="flex-1">
                    <Container title="Password">

                        <div className={`${user?.providerId === "credentials" && "max-w-lg"} cursor-pointer select-none`}
                            onClick={() => openModal(user?.hasPassword ? "changePassword" : "setPassword")}
                        >
                            <ContainerSmall>
                                <div className="flex justify-between">

                                    <div className="flex gap-3 items-center">
                                        <Lock className="w-4 h-4" />
                                        <span className="font-extrabold text-md">{user?.hasPassword ? "Change" : "Set"} Password</span>
                                    </div>

                                    <button className="hover:text-gray-300 transition-all cursor-pointer">
                                        <ChevronRight size={23} strokeWidth={1.2} />
                                    </button>
                                </div>


                            </ContainerSmall>
                        </div>


                    </Container>
                </div>



            </div>


            <Container title="Sessions">

                <div className="w-40 mb-2">
                    <MenuInput leftPart={""} value={showInactive ? "all" : "active"} onChange={(v) => setShowInactive(v === "all")}
                        choices={[{ value: "all", element: "All" }, { value: "active", element: "Active" }]} />
                </div>

                <div className="overflow-x-auto pb-2">
                    <table className="w-full overflow-x-scroll">

                        <TableHead>
                            <TableHeading>Broswer</TableHeading>
                            <TableHeading>Near</TableHeading>
                            <TableHeading>IP Address</TableHeading>
                            <TableHeading>Last Used</TableHeading>
                            <TableHeading>Action</TableHeading>
                        </TableHead>

                        <tbody>
                            {
                                data?.filter((sess) => sess.active).map(sess => <SessionDisplay session={sess} key={sess.id} />)
                            }
                            {
                                data?.filter((sess) => !sess.active).map(sess => <SessionDisplay session={sess} key={sess.id} />)
                            }
                        </tbody>
                    </table>

                </div>
            </Container >
        </div >
    );
}

function SessionDisplay({ session }: { session: Session }) {
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
        <TableRow>
            <TableCell current={session.current}>{(session.browser && session.os) ? `${session.browser} (${session.os})` : "-"}</TableCell>
            <TableCell current={session.current}>{(session.city && session.country) ? `${session.city} (${session.country})` : "-"}</TableCell>
            <TableCell current={session.current}>{session.ip ?? "-"}</TableCell>
            <TableCell current={session.current}>{session.current ? "Online" : timeSince(session.lastUsedAt)}</TableCell>
            <TableCell current={session.current}>
                {session.current ? "Current" : session.active ?
                    <span className="text-red-400 cursor-pointer hover:text-red-300 transition-colors" onClick={() => mutate(session.id)}>Revoke Session</span>
                    : <span className="text-gray-400">Logged out</span>}
            </TableCell>
        </TableRow>
    )
}

