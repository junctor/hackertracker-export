# hackertracker-export

Export HackerTracker events to JSON

## Export Static HackerTracker Data

### Install Dependancies

```bash
    npm install
```

### Export Static Data

```bash
    npm run export
```

_Fetches the 10 recently updated conferences from Firebase and exports static json files to a generated `out` directory_

### Firebase API key

Script requires the Firebase API key to be set as the `FIREBASE_API_KEY` environment variable. This stops @Advice-Dog from getting alerted every time I leak the key, but you are all hackers and undoubtedly you’ll find it anyway.

### Tailwind Safelisting colors

```sh
jq '.[].type.color' ./events.json | sort -u | tr '\n' ',' | sed 's/.$//'
```

[Tailwind Docs](https://tailwindcss.com/docs/content-configuration#safelisting-classes)
