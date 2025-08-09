package security

import internal "go-ormx/ormx/internal/security"

// Re-export selected security API for external usage/tests/examples

type (
	EncryptedField   = internal.EncryptedField
	FieldEncryptor   = internal.FieldEncryptor
	EncryptedString  = internal.EncryptedString
	EncryptionConfig = internal.EncryptionConfig
	ValidationResult = internal.ValidationResult
	SecurityContext  = internal.SecurityContext
)

var (
	NewFieldEncryptor    = internal.NewFieldEncryptor
	NewEncryptedString   = internal.NewEncryptedString
	EncryptSensitiveData = internal.EncryptSensitiveData
	DecryptSensitiveData = internal.DecryptSensitiveData
	ValidateStruct       = internal.ValidateStruct
	NewSecurityContext   = internal.NewSecurityContext
	FromContext          = internal.FromContext
)

func SetDefaultEncryptor(enc *FieldEncryptor) {
	internal.SetDefaultEncryptor((*internal.FieldEncryptor)(enc))
}
