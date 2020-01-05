const Bundler = require('parcel-bundler');
const Path = require('path');
const FS = require('fs');
const Obfuscator = require("javascript-obfuscator");

const module_name = process.argv[2];

if (!module_name) {
  throw new Error('no module name');
}

const entryFiles = Path.join(__dirname, module_name, './index.html');
const outDir = Path.join(__dirname, 'build', module_name);

const options = {
  outDir: outDir,
  cache: false,
  watch: false,
  minify: true,
  hmr: false,
  sourceMaps: false,
  detailedReport: true,
  contentHash: true,
  production: true,
};

const obfuscation = { controlFlowFlattening: true };

(async function () {
  const bundler = new Bundler(entryFiles, options);

  bundler.bundle();

  bundler.on('bundled', async (bundle) => {

    for (let result of bundle.childBundles) {
      if (result.type != 'js') {
        continue;
      }

      await Promise
        .resolve()
        .then(() => new Promise((res, rej) =>
          FS.readFile(result.name, { encoding: 'utf8' }, (err, data) => {
            if (err) {
              return rej(err);
            }

            res(Obfuscator.obfuscate(data, obfuscation).getObfuscatedCode())
          })))
        .then((obfuscated) => new Promise((res, rej) =>
          FS.writeFile(result.name, obfuscated, (err) => {
            if (err) {
              return rej(err);
            }

            res();
          })))
        .then(() => console.log(`→ ${result.name} obfuscated`));
    }
  });
})();