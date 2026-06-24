import { redirect } from "@/lib/i18n/navigation";
import { defaultLocale } from "@/lib/i18n/routing";

export default function RootPage() {
  redirect({ href: "/", locale: defaultLocale });
}
