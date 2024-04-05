import firebaseInit from "./init";
import { getConferences } from "./fb";
import fs from "fs";
import conference from "./conf";

void (async () => {
  const apiKey = process.env.FIREBASE_API_KEY;

  if (apiKey === undefined) {
    console.log("FIREBASE_API_KEY environment variable is not set");
    return;
  }

  const fbDb = await firebaseInit(apiKey);

  const confs = await getConferences(fbDb, 10);

  const outputDir = "./out/ht/";

  fs.mkdirSync(outputDir, { recursive: true });

  await fs.promises.writeFile(`${outputDir}/index.json`, JSON.stringify(confs));

  const confColors = await Promise.all(
    confs
      .filter((conf) => !conf.hidden)
      .map(async (conf) => {
        const result = await conference(fbDb, conf, outputDir);
        return result;
      })
  );

  const allColors = Array.from(
    confColors.reduce((acc, set) => {
      return new Set([...acc, ...set]);
    }, new Set<string>())
  );

  const colorOutput = {
    colors: allColors.sort(),
  };

  await fs.promises.writeFile(
    `${outputDir}/colors.json`,
    JSON.stringify(colorOutput)
  );
})();
