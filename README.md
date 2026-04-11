<div align="center">
  <img src="https://raw.githubusercontent.com/ashleymcnamara/gophers/master/This_is_Fine_Gopher.png" alt="Golang Ticketing System Banner" width="700" />
  <h1>🎟️ High-Concurrency Ticketing System</h1>
  <p>
    <b>A resilient, enterprise-grade Flash Sale ticketing platform.</b><br />
    <i>Engineered with Golang, Redis Lua, Apache Kafka and PostgreSQL.</i>
  </p>
</div>

> Engineered to withstand extreme concurrent traffic spikes (**5,000+ VUs**) with a strict **Zero-Overselling** guarantee, powered by Redis Lua atomic operations and an Event-Driven Kafka architecture.

---

## 🌟 Key Features

* **Zero Overselling:** Utilizes Redis Lua Scripts to guarantee absolute atomicity during ticket deduction, perfectly mitigating Race Conditions during flash sales.

* **Asynchronous Processing (Event-Driven):** Leverages Apache Kafka as a Message Broker to decouple the ticket purchasing flow from database persistence, ensuring extremely low latency for users.

* **Eventual Consistency:** The Kafka Consumer is implemented with a Blocking Retry (Backoff) mechanism, guaranteeing that every successful order is eventually and safely persisted to PostgreSQL.

* **Multi-Layered System Protection:** 
  * **Rate Limiter:** Token Bucket algorithm via Redis Lua and LRU Cache to block spam/DDoS attacks.
  * **Circuit Breaker:** Prevents cascading failures when third-party services degrade.
  * **Sold Out Cache:** In-memory LRU Cache intercepts late requests immediately, drastically reducing database load.

---

## 🛠️ Technology Stack

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Redis](https://img.shields.io/badge/redis-%23DD0031.svg?style=for-the-badge&logo=redis&logoColor=white)
![Apache Kafka](https://img.shields.io/badge/Apache%20Kafka-000?style=for-the-badge&logo=apachekafka)
![PostgreSQL](https://img.shields.io/badge/postgres-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=Prometheus&logoColor=white)
![Grafana](https://img.shields.io/badge/grafana-%23F46800.svg?style=for-the-badge&logo=grafana&logoColor=white)
![Jaeger](https://img.shields.io/badge/Jaeger-%2360D1E3.svg?style=for-the-badge&logo=jaeger&logoColor=white)

* **Backend & API:** Golang (1.21+) | Gin Framework
* **High-Speed Cache & Locking:** Redis (`go-redis`) | Lua Scripts
* **Message Broker:** Apache Kafka (`segmentio/kafka-go`)
* **Persistent Storage:** PostgreSQL (`lib/pq` & `otelsql`)
* **Observability & Tracing:** Jaeger (Distributed Tracing) | Prometheus & Grafana (Metrics Monitoring)
* **Infrastructure & Resiliency:** Docker | `gobreaker` (Circuit Breaker) | `k6` (Spike Testing)

---

## 🚀 Quick Start & Installation

Follow these steps to spin up the entire infrastructure and run the performance benchmarks yourself.

### 1️⃣ Configure Environment Variables

Create a `.env` file in the root directory of the project (you can use `.env.example` as a template) and configure your environment variables:

```env
# POSTGRES
POSTGRES_PORT=YOUR_POSTGRES_PORT
POSTGRES_USER=YOUR_DB_USER
POSTGRES_PASSWORD=YOUR_DB_PASSWORD
POSTGRES_DB=YOUR_DB_NAME

# PGADMIN
PGADMIN_PORT=YOUR_PGADMIN_PORT
PGADMIN_EMAIL=YOUR_PGADMIN_EMAIL
PGADMIN_PASSWORD=YOUR_PGADMIN_PASSWORD

# REDIS
REDIS_PORT=YOUR_REDIS_PORT
REDIS_UI_PORT=YOUR_REDIS_UI_PORT

# ZOOKEEPER
ZOOKEEPER_PORT=YOUR_ZOOKEEPER_PORT

# KAFKA
KAFKA_PORT=YOUR_KAFKA_PORT
KAFKA_INTERNAL_PORT=YOUR_KAFKA_INTERNAL_PORT
KAFKA_UI_PORT=YOUR_KAFKA_UI_PORT

# API SERVICE & WORKER SERVICE
SERVER_PORT=:YOUR_SERVER_PORT
REDIS_ADDR=redis:YOUR_REDIS_PORT
KAFKA_ADDR=kafka:YOUR_KAFKA_INTERNAL_PORT
POSTGRES_ADDR=postgres:YOUR_POSTGRES_PORT
OTEL_EXPORTER_OTLP_ENDPOINT=jaeger:YOUR_JAEGER_OTLP_PORT

# OBSERVABILITY
PROMETHEUS_PORT=YOUR_PROMETHEUS_PORT
GRAFANA_PORT=YOUR_GRAFANA_PORT
GRAFANA_ADMIN_USER=YOUR_GRAFANA_USER
GRAFANA_ADMIN_PASSWORD=YOUR_GRAFANA_PASSWORD
JAEGER_UI_PORT=YOUR_JAEGER_UI_PORT
JAEGER_OTLP_PORT=YOUR_JAEGER_OTLP_PORT
```

### 2️⃣ Spin up Infrastructure

Use Docker Compose to launch PostgreSQL, Redis, Kafka and the application services:
```bash
docker-compose up -d --build
```

### 3️⃣ Initialize Ticket Stock

Before running the load test, you must initialize an event with a specific ticket stock. 

> [!NOTE]
> The number of `Success (200)` responses in the load test is directly controlled by the `stock` value you set here. It is **not** a fixed system limit. For example, if you set `stock: 5000`, the system will process exactly 5,000 successful transactions.

```bash
curl -X POST http://localhost:8080/api/init-ticket \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test",
    "stock": 1000,
    "max_limit": 1000
  }'
```

### 4️⃣ Execute Load Test (k6)

Simulate a massive traffic spike to see the system in action:

```bash
k6 run scripts/k6/buy_ticket.js
```

---

## 📊 Performance Benchmarks

The system was subjected to a rigorous **Spike Test** using `k6` to simulate an extreme Flash Sale scenario: User traffic (VUs) instantly spiked to **5,000 Concurrent Users**.

### 📈 k6 Load Test Report (Extract)

```text
  █ THRESHOLDS 
    ✓ http_req_duration: 'p(95)<1500' p(95)=1.09s

  █ TOTAL RESULTS 
    checks_total.......: 1772304 44251.84/s
    http_reqs......................: 295384  7375.30/s
    vus_max........................: 5000    min=5000 max=5000

  █ HTTP STATUS CODES
    ✗ Success (200)             ↳ ✓ 1000 / ✗ 294384
    ✗ Conflict / Sold Out (409) ↳ ✓ 241668 / ✗ 53716
    ✗ Rate Limited (429)        ↳ ✓ 52716 / ✗ 242668
    ✗ Server Error (500)        ↳ ✓ 0 / ✗ 295384
```

### 🔍 Technical Insights & Metrics

* 🎯 **Absolute Zero Overselling [HTTP 200: 1,000]**
  The system flawlessly processed exactly 1,000 successful transactions out of a 295,000+ request barrage. Strict atomic operations via Redis Lua scripts mathematically guaranteed zero negative inventory or overflow.

* 🛡️ **Flawless Resilience [HTTP 500: 0%]**
  Maintained 100% application uptime under extreme duress (~7,300 req/s). Finely tuned connection pooling and context timeouts completely prevented database starvation and application crashes.

* 🚦 **Proactive Traffic Shaping [HTTP 429 & 409]**
  * **Spam Mitigation (429):** The Token Bucket algorithm surgically intercepted and dropped ~52,000 rapid, abusive requests from identical clients.
  * **Edge Rejection (409):** An L1 in-memory LRU cache activated instantly upon sell-out. It deflected ~241,000 late requests directly at the application gateway, completely shielding the database from catastrophic load.

* ⚡ **Consistent Low Latency [p(95) = 1.09s]**
  Comfortably beat the defined performance thresholds. 95% of the massive incoming traffic—including successful hits, rate limits and rejections—was fully resolved in under 1.09 seconds.

---

## 📜 License

This project is distributed under the **MIT License**. See the [LICENSE](LICENSE) file for more information.
