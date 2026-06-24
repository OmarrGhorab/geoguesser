import { z } from "zod";

const envSchema = z.object({
  BACKEND_API_URL: z.string().url().default("http://localhost:8080/api/v1"),
  NEXT_PUBLIC_APP_URL: z.string().url().default("http://localhost:3000"),
});

export const env = envSchema.parse({
  BACKEND_API_URL: process.env.BACKEND_API_URL,
  NEXT_PUBLIC_APP_URL: process.env.NEXT_PUBLIC_APP_URL,
});
