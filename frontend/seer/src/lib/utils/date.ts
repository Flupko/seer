export function timeSince(date: Date) {
    const seconds = Math.floor((Date.now() - date.getTime()) / 1000);

    const units = [
        { limit: 31536000, name: "year" },
        { limit: 2592000, name: "month" },
        { limit: 86400, name: "day" },
        { limit: 3600, name: "hour" },
        { limit: 60, name: "minute" },
    ] as const;

    for (const u of units) {
        const count = Math.floor(seconds / u.limit);
        if (count >= 1) return `${count} ${u.name}${count === 1 ? "" : "s"} ago`;
    }
    const s = Math.floor(seconds);
    return `${s} second${s === 1 ? "" : "s"} ago`;
}
