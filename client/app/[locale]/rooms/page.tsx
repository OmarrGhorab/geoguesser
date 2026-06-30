import { getTranslations } from "next-intl/server";
import { CreateJoinForm } from "@/features/rooms/create-join-form";

type RoomsPageProps = {
  params: Promise<{
    locale: string;
  }>;
};

export default async function RoomsPage({ params }: RoomsPageProps) {
  const { locale } = await params;
  const t = await getTranslations("Rooms");

  return (
    <CreateJoinForm
      locale={locale}
      labels={{
        title: t("title"),
        create: t("create"),
        join: t("join"),
        mapId: t("mapId"),
        roomCode: t("roomCode"),
        displayName: t("displayName"),
      }}
    />
  );
}
