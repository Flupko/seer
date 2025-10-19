"use client"

import { getSessions } from "@/lib/api";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useModalStore } from "@/lib/stores/modal";
import ContainerSmall from "@/ui/containers/ContainerSmall";
import MenuInput from "@/ui/menu_small_vertical/MenuVertical";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight } from "lucide-react";
import { useState } from "react";
import Google from '../../../../public/google.svg';
import Lock from "../../../../public/lock.svg";
import Twitch from '../../../../public/twitch.svg';
import Container from "../Container";
import SessionDisplay from "./Session";

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

                        <thead className="h-12 border-separate border-spacing-0">
                            <tr className="text-gray-200 text-sm text-left">
                                <TableHeading>Broswer</TableHeading>
                                <TableHeading>Near</TableHeading>
                                <TableHeading>IP Address</TableHeading>
                                <TableHeading>Last Used</TableHeading>
                                <TableHeading>Action</TableHeading>
                            </tr>
                        </thead>

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
            </Container>
        </div>
    );
}

function TableHeading({ children }: { children: React.ReactNode }) {
    return (
        <th className="text-gray-200 text-sm text-left font-bold px-6 text-nowrap">
            {children}
        </th>
    )
}