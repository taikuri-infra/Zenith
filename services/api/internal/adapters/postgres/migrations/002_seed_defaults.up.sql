-- Seed default platform settings
INSERT INTO platform_settings (id, platform_name, base_domain, provider, default_region, region_label, auto_backups, retention_days)
VALUES (1, 'Zenith', 'freezenith.com', 'Hetzner Cloud', 'fsn1', 'Falkenstein', true, 30)
ON CONFLICT (id) DO NOTHING;

-- Seed default modules
INSERT INTO modules (name, installed, latest, status, description) VALUES
    ('Zenith Operator',  'v1.2.1', 'v1.3.0', 'update_available', 'Core platform operator'),
    ('CloudNativePG',    'v1.22.1','v1.23.0','update_available', 'PostgreSQL operator'),
    ('Redis Operator',   'v7.2.0', 'v7.2.0', 'up_to_date',      'Redis operator'),
    ('cert-manager',     'v1.14.2','v1.14.2','up_to_date',      'SSL certificate management'),
    ('Traefik',          'v2.11.0','v2.11.0','up_to_date',      'Ingress controller'),
    ('Harbor',           'v2.10.0','v2.10.1','update_available', 'Container registry'),
    ('Keycloak Operator','v24.0.0','v24.0.0','up_to_date',      'Identity & access management'),
    ('Prometheus Stack', 'v56.2.0','v56.2.0','up_to_date',      'Monitoring & alerting'),
    ('Loki',             'v3.0.1', 'v3.0.1', 'up_to_date',      'Log aggregation'),
    ('NATS',             'v2.10.0','v2.10.0','up_to_date',      'Message queue & KV store'),
    ('Linkerd',          'v2.14.0','v2.14.1','update_available', 'Service mesh')
ON CONFLICT (name) DO NOTHING;

-- Seed update history
INSERT INTO update_history (version, date, status) VALUES
    ('v1.2.1', '2026-01-15', 'installed'),
    ('v1.2.0', '2025-12-20', 'superseded'),
    ('v1.1.0', '2025-11-01', 'superseded')
ON CONFLICT (version) DO NOTHING;
