# HackerTracker Export

A tool to export HackerTracker events to JSON format.

## Table of Contents

- [Introduction](#introduction)
- [Installation](#installation)
- [Usage](#usage)
  - [Export Static Data](#export-static-data)
  - [Tailwind Safelisting Colors](#tailwind-safelisting-colors)
- [References](#references)

## Introduction

HackerTracker Export is a utility designed to fetch and export the most recently updated HackerTracker events from Firebase into static JSON files.

## Installation

### Install Dependencies

To get started, install the required npm packages:

```bash
npm install
```

## Usage

### Export Static Data

To export the static data, run the following command:

```bash
npm run export
```

This command will fetch the 25 most recently updated conferences from Firebase and export them as static JSON files into a newly generated `out` directory.

### Tailwind Safelisting Colors

To safelist colors for Tailwind CSS, use the following command:

```sh
jq '.[].type.color' ./events.json | sort -u | tr '\n' ',' | sed 's/.$//'
```

For more information on safelisting classes in Tailwind CSS, refer to the [Tailwind CSS Documentation](https://tailwindcss.com/docs/content-configuration#safelisting-classes).

## References

- [HackerTracker](https://hackertracker.app/)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs/)
