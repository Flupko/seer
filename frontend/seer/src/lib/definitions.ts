import { Decimal } from "decimal.js";
import { z } from "zod";
import { pricesForMarket } from "./lslmsr/lslmsr";


const usernameSchema = z
  .string()
  .min(3, { message: "Username must be at least 3 characters" })
  .max(15, { message: "Username must be at most 15 characters" })
  .regex(/^[a-z0-9]+$/i, { message: "Username must be alphanumeric (A–Z, a-z, 0–9) only" });

const emailSchema = z.string().min(1, "Email is required").email({ message: "Email is invalid" });

const statusSchema = z.enum(['pending_email_verification', 'activated', 'credentials']);

const providerSchema = z.enum(['credentials', 'google', 'twitch']).optional();

const passwordSchema = z
  .string()
  .min(8, "Password must be at least 8 characters")
  .max(49, "Password must be at most 49 characters");



export const UserSchema = z.object({
  id: z.uuid(),
  providerId: providerSchema,
  hasPassword: z.boolean(),
  email: emailSchema,
  username: usernameSchema,
  profileImageUrl: z.url().optional(),
  status: statusSchema,
});

export type User = z.infer<typeof UserSchema>;

export const RegisterSchema = z.object({
  username: usernameSchema,
  email: emailSchema,
  password: passwordSchema,
});


export const LoginSchema = z.object({
  login: z.union([emailSchema, usernameSchema]),
  password: z.string().min(1, "Password is required"),
});

export const ProfileCompletionSchema = z.object({
  username: usernameSchema,
});

export const SessionSchema = z.object({
  id: z.uuid(),
  lastUsedAt: z.coerce.date(),
  os: z.string().optional(),
  browser: z.string().optional(),
  device: z.string().optional(),
  ip: z.string().optional(),
  country: z.string().optional(),
  city: z.string().optional(),
  active: z.boolean(),
  current: z.boolean(),
});

export const SessionsSchema = z.array(SessionSchema);

export const ChangePasswordSchema = z.object({
  currentPassword: z.string().min(1, "Current Password is required"),
  newPassword: passwordSchema.min(8, "New password must be at least 8 characters")
    .max(49, "New password must be at most 49 characters"),
  confirmNewPassword: z.string().min(1, "Please confirm your new password")
}).refine((data) => data.newPassword === data.confirmNewPassword, {
  message: "Passwords don't match",
  path: ['confirmNewPassword'],
});

export const SetPasswordSchema = z.object({
  password: passwordSchema,
  confirmPassword: z.string().min(1, "Please confirm your password")
}).refine((data) => data.password === data.confirmPassword, {
  message: "Passwords don't match",
  path: ['confirmPassword'],
});

export const UserPreferencesSchema = z.object({
  hidden: z.boolean(),
  receiveMarketingEmails: z.boolean(),
});



export const ChangePasswordPayloadSchema = ChangePasswordSchema.omit({ confirmNewPassword: true });
export const SetPasswordPayloadSchema = SetPasswordSchema.omit({ confirmPassword: true });

export type RegisterFormValues = z.infer<typeof RegisterSchema>;
export type LoginFormValues = z.infer<typeof LoginSchema>;
export type ProfileCompletionFormValues = z.infer<typeof ProfileCompletionSchema>;
export type Session = z.infer<typeof SessionSchema>;
export type Sessions = z.infer<typeof SessionsSchema>;

export type ChangePasswordFormValues = z.infer<typeof ChangePasswordSchema>
export type ChangePasswordPayload = z.infer<typeof ChangePasswordPayloadSchema>

export type SetPasswordFormValues = z.infer<typeof SetPasswordSchema>
export type SetPasswordPayload = z.infer<typeof SetPasswordPayloadSchema>

export type UserPreferences = z.infer<typeof UserPreferencesSchema>;
export type UpdateUserPreferences = Partial<UserPreferences>;



export const CategorySchema = z.object({
  id: z.number(),
  slug: z.string(),
  label: z.string(),
  iconUrl: z.string(),
});

export type Category = z.infer<typeof CategorySchema>;


export const MarketSort = z.enum(['trending', 'volume', 'newest', 'endingSoon']);
export const MarketStatus = z.enum(['active', 'resolved']);

export const MarketSearchSchema = z.object({
  query: z.string().min(3).max(50).optional(),
  categorySlug: z.string().min(1).max(20).optional(),
  status: MarketStatus.default('active'),
  sort: MarketSort.default('trending'),
  pageSize: z.number().int().min(3).max(20).default(20),
  page: z.number().int().min(1).default(1),
});

// Custom Decimal Schema
export const DecimalSchema = z
  .union([
    z.string().trim().min(1), // allow strings
    z.number(), // allow JS numbers
    z.instanceof(Decimal),// already a Decimal
  ])
  .transform((val, ctx) => {
    try {
      if (val instanceof Decimal) return val;
      return new Decimal(val as string | number);
    } catch {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: "Invalid decimal input" });
      return z.NEVER;
    }
  });

export type MarketSearch = z.infer<typeof MarketSearchSchema>;

export const OutcomeSchema = z.object({
  id: z.number(),
  name: z.string(),
  quantity: DecimalSchema,
  position: z.number().int(),
  price: DecimalSchema.default(new Decimal(0)),
});

export type Outcome = z.infer<typeof OutcomeSchema>;

export const MarketViewSchema = z.object({
  id: z.uuid(),
  name: z.string(),
  description: z.string().nullable(),
  imgKey: z.string().optional(),
  slug: z.string(),
  closeTime: z.string().optional(),
  outcomeSort: z.enum(['price', 'position']),
  alpha: DecimalSchema,
  fee: DecimalSchema,
  capPrice: DecimalSchema,
  categories: z.array(CategorySchema),
  outcomes: z.array(OutcomeSchema),
  version: z.number().int().default(0),
}).transform((market) => {
  // Compute prices for each outcome
  pricesForMarket(market);
  return market
});

export type MarketView = z.infer<typeof MarketViewSchema>;

export const MetadataSchema = z.object({
  currentPage: z.number().int().min(1),
  pageSize: z.number().int().min(1),
  firstPage: z.number().int().min(1),
  lastPage: z.number().int().min(1),
  totalRecords: z.number().int().min(0),
});

export type Metadata = z.infer<typeof MetadataSchema>;

export const CurrencySchema = z.enum(["USDT"]);
export type Currency = z.infer<typeof CurrencySchema>;

export const BalanceSchema = z.object({
  balance: DecimalSchema,
  currency: CurrencySchema,
  version: z.number().int(),
});

export type Balance = z.infer<typeof BalanceSchema>;

export const PlaceBetSchema = z.object({
  betAmount: DecimalSchema,
  minWantedGain: DecimalSchema,
  marketId: z.uuid(),
  outcomeId: z.number().int(),
  currency: CurrencySchema,
  idempotencyKey: z.string().min(1).max(36),
});

export type PlaceBet = z.infer<typeof PlaceBetSchema>;



export const sortOptions = [
  { value: "trending", element: "Trending" },
  { value: "volume", element: "Volume" },
  { value: "newest", element: "Newest" },
  { value: "endingSoon", element: "Ending Soon" },
];

export const statusOptions = [
  { value: "active", element: "Active" },
  { value: "resolved", element: "Resolved" },
];

export const QuoteFormSchema = z.object({
  wagerUsd: z.number().min(1, "Wager must be at least $1").max(10000, "Wager must be at most $10,000"),
})