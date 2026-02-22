# Capability: Storage Management

## Purpose
Manage S3-compatible object storage buckets scoped to projects with private or public-read access.

## Requirements

### Requirement: Object Storage Buckets
The system SHALL support CRUD for S3-compatible object storage buckets scoped to a project. Buckets can be `private` or `public-read`.

#### Scenario: Create private bucket
- **WHEN** a user POSTs to `/api/v1/projects/:id/storage` with access `private`
- **THEN** a private object storage bucket is created

#### Scenario: Create public bucket
- **WHEN** a user POSTs with access `public-read`
- **THEN** a publicly readable bucket is created

### Requirement: Bucket Listing and Detail
The system SHALL support listing all buckets per project and getting individual bucket details including name, objects count, size, access level, and region.

#### Scenario: List buckets
- **WHEN** a user GETs `/api/v1/projects/:id/storage`
- **THEN** all buckets for the project are returned

#### Scenario: Delete bucket
- **WHEN** a user DELETEs `/api/v1/projects/:id/storage/:name`
- **THEN** the bucket is removed
