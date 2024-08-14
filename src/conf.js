import fs from "fs";
import { getEvents, getSpeakers, getTags } from "./fb.js";

export default async function conference(fbDb, htConf, outputDir) {
  const childDir = `${outputDir}/conferences/${htConf.code}`;

  fs.mkdirSync(childDir, { recursive: true });

  const [htEvents, htSpeakers, htTags] = await Promise.all([
    getEvents(fbDb, htConf.code),
    getSpeakers(fbDb, htConf.code),
    getTags(fbDb, htConf.code),
  ]);

  await Promise.all([
    fs.promises.writeFile(`${childDir}/events.json`, JSON.stringify(htEvents)),
    fs.promises.writeFile(
      `${childDir}/speakers.json`,
      JSON.stringify(htSpeakers)
    ),
    fs.promises.writeFile(`${childDir}/tags.json`, JSON.stringify(htTags)),
  ]);

  const eventColors = htEvents.map((e) => e.type.color);
  const tagColors = htTags.flatMap((t) =>
    t.tags.map((e) => e.color_background)
  );

  return new Set(eventColors, tagColors);
}
