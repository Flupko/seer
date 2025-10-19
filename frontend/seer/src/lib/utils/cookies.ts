import { cookies } from "next/headers";

export async function getServerCookies() {
    const cookieStore = await cookies();
    return cookieStore
        .getAll()
        .map(({ name, value }) => `${name}=${value}`)
        .join("; ");
}