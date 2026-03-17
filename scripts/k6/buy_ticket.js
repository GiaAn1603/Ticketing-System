import http from "k6/http";
import { check } from "k6";

export let options = {
  stages: [
    { duration: "20s", target: 1000 },
    { duration: "40s", target: 5000 },
    { duration: "10s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<1000"],
  },
};

export default function () {
  const userID = `user_${__VU}`;
  const requestID = `req-${__VU}-${__ITER}`;
  const octet3 = Math.floor(__VU / 250);
  const octet4 = __VU % 250;
  const fakeClientIP = `10.0.${octet3}.${octet4}`;

  const payload = JSON.stringify({
    event_id: "test",
    user_id: userID,
    quantity: 1,
  });

  const params = {
    headers: {
      "Content-Type": "application/json",
      "X-Request-Id": requestID,
      "X-Forwarded-For": fakeClientIP,
      "X-Real-IP": fakeClientIP,
    },
  };

  const result = http.post("http://localhost:8080/buy-ticket", payload, params);

  check(result, {
    "Success (200)": (r) => r.status === 200,
    "Bad Request (400)": (r) => r.status === 400,
    "Not Found (404)": (r) => r.status === 404,
    "Conflict / Sold Out (409)": (r) => r.status === 409,
    "Rate Limited (429)": (r) => r.status === 429,
    "Server Error (500)": (r) => r.status === 500,
  });
}
