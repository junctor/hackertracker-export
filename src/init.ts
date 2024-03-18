import { type Firestore } from "firebase/firestore";
import { initializeApp } from "firebase/app";
import { getFirestore } from "firebase/firestore/lite";
import { firebaseConfig } from "./ config";

export default async function firebaseInit(apiKey: string): Promise<Firestore> {
  const config = firebaseConfig(apiKey);
  const app = initializeApp(config);
  const db = getFirestore(app);
  return db;
}
