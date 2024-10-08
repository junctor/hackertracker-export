import firebaseInit from "./init.js";
import { getConferences } from "./fb.js";
import fs from "fs";
import conference from "./conf.js";

void (async () => {
  const fbDb = await firebaseInit();

  const confs = await getConferences(fbDb, 25);

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
    }, new Set())
  );

  const colorOutput = {
    colors: allColors.sort(),
  };

  await fs.promises.writeFile(
    `${outputDir}/colors.json`,
    JSON.stringify(colorOutput)
  );
})();
