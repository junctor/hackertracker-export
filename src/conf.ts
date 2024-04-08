import type { Firestore } from "firebase/firestore";
import fs from "fs";
import { getEvents } from "./fb";

export default async function conference(
  fbDb: Firestore,
  htConf: HTConference,
  outputDir: string
): Promise<Set<string>> {
  const childDir = `${outputDir}/conferences/${htConf.code}`;

  fs.mkdirSync(childDir, { recursive: true });

  const [htEvents] = await Promise.all([getEvents(fbDb, htConf.code)]);

  await Promise.all([
    fs.promises.writeFile(`${childDir}/events.json`, JSON.stringify(htEvents)),
  ]);

  const eventColors = new Set(htEvents.map((e) => e.type.color as string));

  return eventColors;
}
