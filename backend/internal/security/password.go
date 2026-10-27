package security

import "github.com/alexedwards/argon2id"

const (
	PasswordMemoryKiB               = 19 * 1024
	PasswordIterations              = 2
	PasswordParallelism             = 1
	PasswordSaltBytes               = 16
	PasswordKeyBytes                = 32
	MinimumPasswordLength           = 12
	MinimumSystemPasswordCharacters = 15
	MaximumPasswordLength           = 128
	MaxConcurrentPasswordChecks     = 4
	MaximumUsernameLength           = 120
)

func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, &argon2id.Params{Memory: PasswordMemoryKiB, Iterations: PasswordIterations, Parallelism: PasswordParallelism, SaltLength: PasswordSaltBytes, KeyLength: PasswordKeyBytes})
}
func VerifyPassword(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}
