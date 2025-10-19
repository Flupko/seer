import { Metadata } from "@/lib/definitions";

export function getNextPageParamFromMetadata(meta: Metadata): number | undefined {
    return meta.currentPage < meta.lastPage ? meta.currentPage + 1 : undefined;
}