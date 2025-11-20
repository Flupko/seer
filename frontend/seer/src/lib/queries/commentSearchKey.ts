import { CommentSearch } from "../definitions";

export const commentSearchKey = (search: Omit<CommentSearch, 'pageSize' | 'page'>) => ['comment', 'marketId', search.marketId, 'parentId', search.parentId ?? 'root'];