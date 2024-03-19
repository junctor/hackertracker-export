/* eslint-disable @typescript-eslint/explicit-function-return-type */
import {
  collection,
  getDocs,
  query,
  orderBy,
  limit,
} from "firebase/firestore/lite";
import { type Firestore } from "firebase/firestore";

export async function getSpeakers(db: Firestore, conference: string) {
  const docRef = collection(db, "conferences", conference, "speakers");
  const docSnap = await getDocs(docRef);
  const firebaseData = docSnap.docs.map(
    (speakerDoc) => speakerDoc.data() as any
  );

  return firebaseData;
}

export async function getConferences(
  db: Firestore,
  count: number = 10
): Promise<HTConference[]> {
  const docRef = collection(db, "conferences");
  const q = query(docRef, orderBy("updated_at", "desc"), limit(count));
  const docSnap = await getDocs(q);
  const firebaseData = docSnap.docs.map((eventsDoc: { data: () => any }) =>
    eventsDoc.data()
  );

  return firebaseData;
}

export async function getEvents(db: Firestore, conference: string) {
  const docRef = collection(db, "conferences", conference, "events");
  const q = query(docRef, orderBy("begin_timestamp", "desc"));
  const docSnap = await getDocs(q);
  const firebaseData = docSnap.docs.map((eventsDoc: { data: () => any }) =>
    eventsDoc.data()
  );

  return firebaseData;
}
