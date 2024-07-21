import { initializeApp } from "firebase/app";
import { getFirestore } from "firebase/firestore/lite";
import { firebaseConfig } from "./config.js";

export default async function firebaseInit(apiKey) {
  const config = firebaseConfig(apiKey);
  const app = initializeApp(config);
  const db = getFirestore(app);
  return db;
}
