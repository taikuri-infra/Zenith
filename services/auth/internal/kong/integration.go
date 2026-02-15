package kong

// KongConfig represents the JWT plugin configuration for Kong
type KongConfig struct {
	RealmID    string
	JWTSecret  string
	Issuer     string
	ConsumerID string
}

// JWTPluginConfig generates Kong JWT plugin configuration for a realm
func JWTPluginConfig(realmID, issuer, jwtSecret string) map[string]interface{} {
	return map[string]interface{}{
		"name": "jwt",
		"config": map[string]interface{}{
			"key_claim_name":     "iss",
			"secret_is_base64":   false,
			"claims_to_verify":   []string{"exp"},
			"header_names":       []string{"Authorization"},
			"maximum_expiration": 3600,
		},
	}
}

// ConsumerConfig generates a Kong consumer for a realm client
func ConsumerConfig(realmID, clientID string) map[string]interface{} {
	return map[string]interface{}{
		"username":  realmID + "-" + clientID,
		"custom_id": clientID,
		"tags":      []string{"zenith", "realm:" + realmID},
	}
}

// JWTCredentialConfig generates JWT credentials for a consumer
func JWTCredentialConfig(issuer, jwtSecret string) map[string]interface{} {
	return map[string]interface{}{
		"key":       issuer,
		"secret":    jwtSecret,
		"algorithm": "HS256",
	}
}
