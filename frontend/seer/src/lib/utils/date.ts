export function timeSince(date: Date) {
    const seconds = Math.floor((Date.now() - date.getTime()) / 1000);

    if (seconds == 0) return "now";

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


export function formatDateTime(date: Date): string {
    const now = new Date();
    const diffMs = date.getTime() - now.getTime();
    const diffSec = Math.round(diffMs / 1000);
    const diffMin = Math.round(diffMs / 60000);
    const diffHours = Math.round(diffMs / 3600000);
    const diffDays = Math.round(diffMs / 86400000);
    const diffMonths = Math.round(diffMs / 2592000000); // ~30 days

    // Format the date part: "Nov 03, 4:20 PM"
    const dateFormatter = new Intl.DateTimeFormat('en-US', {
        month: 'short',
        day: '2-digit',
        hour: 'numeric',
        minute: '2-digit',
        hour12: true
    });

    const formattedDate = dateFormatter.format(date);

    // Format the relative time part
    const rtf = new Intl.RelativeTimeFormat('en', {
        numeric: 'auto'
    });

    let relativeTime;
    const absDiffMin = Math.abs(diffMin);
    const absDiffHours = Math.abs(diffHours);
    const absDiffDays = Math.abs(diffDays);
    const absDiffMonths = Math.abs(diffMonths);

    if (absDiffMin < 60) {
        relativeTime = rtf.format(diffMin, 'minute');
    } else if (absDiffHours < 24) {
        relativeTime = rtf.format(diffHours, 'hour');
    } else if (absDiffDays < 30) {
        relativeTime = rtf.format(diffDays, 'day');
    } else {
        relativeTime = rtf.format(diffMonths, 'month');
    }

    return `${formattedDate} - ${relativeTime}`;
}