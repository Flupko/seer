import Image from "next/image";

export default function ProfilePicture({ url, size }: { url?: string, size: number }) {
    return (
        <Image src={url ?? "https://api.dicebear.com/9.x/glass/png"} width={size} height={size} alt="profile image" className="rounded-full shadow-2xl" />
    )
}