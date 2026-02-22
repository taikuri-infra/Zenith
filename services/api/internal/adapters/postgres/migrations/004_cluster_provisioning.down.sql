DROP INDEX IF EXISTS idx_customers_capi_cluster;

ALTER TABLE customers
  DROP COLUMN IF EXISTS capi_cluster_name,
  DROP COLUMN IF EXISTS cluster_region,
  DROP COLUMN IF EXISTS cluster_nodes,
  DROP COLUMN IF EXISTS cluster_k8s_version;
