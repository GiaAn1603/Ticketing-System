CREATE TABLE IF NOT EXISTS orders (
  id SERIAL PRIMARY KEY,
  event_id VARCHAR(50) NOT NULL,
  user_id VARCHAR(50) NOT NULL,
  request_id VARCHAR(50) NOT NULL,
  quantity INT NOT NULL DEFAULT 1,
  status VARCHAR(20) NOT NULL DEFAULT 'Success',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_event_user_req UNIQUE (event_id, user_id, request_id)
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
