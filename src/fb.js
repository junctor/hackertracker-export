/* eslint-disable @typescript-eslint/explicit-function-return-type */
import {
  collection,
  getDocs,
  query,
  orderBy,
  limit,
} from "firebase/firestore/lite";

export async function getConferences(db, count = 10) {
  const docRef = collection(db, "conferences");
  const q = query(docRef, orderBy("updated_at", "desc"), limit(count));
  const docSnap = await getDocs(q);
  const firebaseData = docSnap.docs.map((eventsDoc) => eventsDoc.data());

  return firebaseData;
}

export async function getEvents(db, conference) {
  const docRef = collection(db, "conferences", conference, "events");
  const q = query(docRef, orderBy("begin_timestamp", "desc"));
  const docSnap = await getDocs(q);
  const firebaseData = docSnap.docs.map((eventsDoc) => eventsDoc.data());

  return firebaseData;
}
