ALTER TABLE customers
  ADD COLUMN capi_cluster_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN cluster_region TEXT NOT NULL DEFAULT 'fsn1',
  ADD COLUMN cluster_nodes INTEGER NOT NULL DEFAULT 3,
  ADD COLUMN cluster_k8s_version TEXT NOT NULL DEFAULT 'v1.31.2';

CREATE INDEX idx_customers_capi_cluster ON customers(capi_cluster_name) WHERE capi_cluster_name != '';
