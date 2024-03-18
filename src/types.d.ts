interface Config {
  apiKey: string;
  authDomain: string;
  databaseURL: string;
  projectId: string;
  messagingSenderId: string;
  appId: string;
  measurementId: string;
}

interface Timestamp {
  seconds: number;
  nanoseconds: number;
}

interface HTMaps {
  description: string;
  file: string;
  filename: string;
  id: number;
  name_text: string;
  name: string;
  sort_order: number;
  url: string;
}

interface HTConference {
  code: string;
  codeofconduct?: string;
  conference_id: number;
  description: string;
  enable_merch: boolean;
  end_date: string;
  end_timestamp_str: string;
  end_timestamp: Timestamp;
  hidden: boolean;
  hidden: false;
  id: number;
  kickoff_timestamp_str: string;
  kickoff_timestamp: Timestamp;
  link: string;
  maps: HTMaps[];
  name: string;
  start_date: string;
  start_timestamp_str: string;
  start_timestamp: Timestamp;
  supportdoc: string | null;
  tagline_text: string;
  timezone: string;
  updated_at: Timestamp;
  enable_merch_cart: boolean;
  kickoff_timestamp_str: string;
  updated_at: Timestamp;
}
