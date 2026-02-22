package entities

// BucketAccess represents the access policy of a storage bucket.
type BucketAccess string

const (
	BucketAccessPrivate BucketAccess = "private"
	BucketAccessPublic  BucketAccess = "public"
)

// BucketStatus represents the lifecycle status of a storage bucket.
type BucketStatus string

const (
	BucketStatusCreating BucketStatus = "creating"
	BucketStatusActive   BucketStatus = "active"
	BucketStatusError    BucketStatus = "error"
	BucketStatusDeleting BucketStatus = "deleting"
)

// UserBucket represents an S3-compatible storage bucket provisioned for a user's app.
type UserBucket struct {
	ID        string       `json:"id"`
	AppID     string       `json:"app_id"`
	UserID    string       `json:"user_id"`
	Name      string       `json:"name"`
	Access    BucketAccess `json:"access"`
	Region    string       `json:"region"`
	SizeMB    int          `json:"size_mb"`
	MaxSizeMB int          `json:"max_size_mb"`
	Objects   int          `json:"objects"`
	Status    BucketStatus `json:"status"`
	Endpoint  string       `json:"endpoint"`
	Timestamps
}
