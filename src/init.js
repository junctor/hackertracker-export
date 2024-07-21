import { initializeApp } from "firebase/app";
import { getFirestore } from "firebase/firestore/lite";
import { firebaseConfig as config } from "./config.js";

export default async function firebaseInit() {
  const app = initializeApp(config);
  const db = getFirestore(app);
  return db;
}
