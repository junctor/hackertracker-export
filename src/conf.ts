import type { Firestore } from "firebase/firestore";
import fs from "fs";
import { getEvents, getSpeakers, getLocations, getTags } from "./fb";

export default async function conference(
  fbDb: Firestore,
  htConf: HTConference,
  outputDir: string
): Promise<void> {
  const childDir = `${outputDir}/conferences/${htConf.code}`;

  fs.mkdirSync(childDir, { recursive: true });

  const [htEvents, htSpeakers, htFAQ, htLocations] = await Promise.all([
    getEvents(fbDb, htConf.code),
    getSpeakers(fbDb, htConf.code),
    getLocations(fbDb, htConf.code),
    getTags(fbDb, htConf.code),
  ]);

  await Promise.all([
    fs.promises.writeFile(
      `${childDir}/conference.json`,
      JSON.stringify(htConf)
    ),
    fs.promises.writeFile(`${childDir}/events.json`, JSON.stringify(htEvents)),
    fs.promises.writeFile(
      `${childDir}/speakers.json`,
      JSON.stringify(htSpeakers)
    ),
    fs.promises.writeFile(`${childDir}/faq.json`, JSON.stringify(htFAQ)),
    fs.promises.writeFile(
      `${childDir}/locations.json`,
      JSON.stringify(htLocations)
    ),
  ]);
}
