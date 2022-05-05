#!/usr/bin/env node

const fs = require('fs');
const semver = require('semver');
const yargs = require('yargs');
const yaml = require('js-yaml');
const merge = require('lodash.merge');
const { hideBin } = require('yargs/helpers');
const { InstallerVersions } = require('../../web/src/installers/versions');

const specDir = './addons';

var getImages = () => {
    const images = [];
    fs.readdirSync(specDir).forEach((addon) => {
        const addonDir = `${specDir}/${addon}`;
        const stats = fs.statSync(addonDir);
        if (!stats.isDirectory()) {
            return;
        }
        fs.readdirSync(addonDir).forEach((version) => {
            const versionDir = `${specDir}/${addon}/${version}`;
            const stats = fs.statSync(versionDir);
            if (!stats.isDirectory()) {
                return;
            }
            const manifestFile = `${specDir}/${addon}/${version}/Manifest`;
            if (!fs.existsSync(manifestFile)) {
                return;
            }
            const grypeConfigFile = `./bin/scan-images/.grype.yaml`;
            let grypeConfig = yaml.load(fs.readFileSync(grypeConfigFile, 'utf-8'));
            const grypeConfigFileOverride = `${specDir}/${addon}/${version}/.grype.yaml`;
            if (fs.existsSync(grypeConfigFileOverride)) {
                const grypeConfigOverride = yaml.load(fs.readFileSync(grypeConfigFileOverride, 'utf-8'));
                grypeConfig = merge(grypeConfig, grypeConfigOverride);
            }
            const grypeConfigBase64 = Buffer.from(yaml.dump(grypeConfig)).toString('base64');
            fs.readFileSync(manifestFile, 'utf-8').split(/\r?\n/).forEach((line) => {
                const parts = line.split(' ');
                if (parts[0] !== 'image') {
                    return;
                }
                const image = {
                    addon: addon,
                    version: version,
                    name: parts[1],
                    image: parts[2],
                    grypeConfig: grypeConfigBase64,
                };
                images.push(image);
            });
        });
    });
    return images;
};

var findLatestAddonVersions = () => {
    const versions = {};
    fs.readdirSync(specDir).forEach((addon) => {
        const addonDir = `${specDir}/${addon}`;
        const stats = fs.statSync(addonDir);
        if (!stats.isDirectory()) {
            return;
        }

        versions[addon] = [];

        let latestVersion = '';
        if (addon in InstallerVersions) {
            if (InstallerVersions[addon].includes('alpha')) {
                latestVersion = 'alpha';
            } else {
                latestVersion = InstallerVersions[addon][0];
            }
            versions[addon].push(latestVersion);
        }

        // this loop finds the greatest version and adds it if it is not in the latest spec
        let greatestVersion = '';
        fs.readdirSync(addonDir).some((version) => {
            const versionDir = `${specDir}/${addon}/${version}`;
            const stats = fs.statSync(versionDir);
            if (!stats.isDirectory()) {
                return false;
            }
            const manifestFile = `${specDir}/${addon}/${version}/Manifest`;
            if (!fs.existsSync(manifestFile)) {
                return false;
            }
            if (semver.valid(version)) {
                if (!greatestVersion || semver.gte(version, greatestVersion)) {
                    greatestVersion = version;
                }
            } else if (version > greatestVersion) {
                greatestVersion = version;
            }
        });

        if (greatestVersion && latestVersion != greatestVersion) {
            versions[addon].push(greatestVersion);
        }
    });
    return versions;
};

var matrix = () => {
    const images = getImages();
    const addonVersions = findLatestAddonVersions();
    const filteredImages = images.filter((image) => {
        return addonVersions[image.addon].some((addonVersion) => {
            return addonVersion === image.version;
        });
    });
    console.log(JSON.stringify({include: filteredImages})); // format for git
};

yargs(hideBin(process.argv))
    .command('$0', 'build images matrix', () => {
        matrix();
    })
    .argv;
