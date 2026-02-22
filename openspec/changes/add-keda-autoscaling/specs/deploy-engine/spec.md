## ADDED Requirements

### Requirement: KEDA Scale-to-Zero for Free Tier
The system SHALL create KEDA `ScaledObject` and `HTTPScaledObject` resources for free-tier apps, scaling them to zero replicas after 15 minutes of inactivity. Pro/Team/Enterprise apps SHALL remain always-on.

#### Scenario: Free-tier app sleeps after inactivity
- **WHEN** a free-tier app receives no HTTP traffic for 15 minutes
- **THEN** KEDA scales the app to 0 replicas (sleeping state)

#### Scenario: Sleeping app wakes on request
- **WHEN** an HTTP request hits a sleeping app
- **THEN** KEDA HTTP Add-on intercepts the request, scales to 1 replica, and proxies the request (target: <5s latency)

#### Scenario: Pro app never sleeps
- **WHEN** a Pro/Team/Enterprise app has no traffic
- **THEN** the app remains running (no ScaledObject created)
