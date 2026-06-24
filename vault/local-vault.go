package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"golang.org/x/crypto/argon2"
	"gopkg.in/yaml.v3"
)

const V1String = "cg:local:v1"

type LocalVault struct {
	Vault LocalVaultFile
}

type LocalVaultFile struct {
	Version      string                 `yaml:"version"`
	CryptoConfig LocalVaultCryptoConfig `yaml:"config"`
	Data         LocalVaultData         `yaml:"data"`
}

type LocalVaultCryptoConfig struct {
	Salt  HexBytes `yaml:"salt"`
	Key   HexBytes `yaml:"key"`
	Nonce HexBytes `yaml:"nonce"`
}

type LocalVaultData struct {
	Ciphertext HexBytes `yaml:"ciphertext"`
	Nonce      HexBytes `yaml:"nonce"`
}

func (lvf LocalVaultFile) GetAad() []byte {
	return []byte(fmt.Sprintf("%s", lvf.Version))
}

func (lv *LocalVault) WriteData(passphrase []byte, data []byte) error {
	derived_key_encryption_key := derive_kek_from_passphrase(
		passphrase,
		lv.Vault.CryptoConfig.Salt,
	)

	// this is just to ensure the same passphrase is used
	_, err := lv.ReadData(passphrase)

	if err != nil {
		return err
	}

	new_dek := make_key()
	new_dek_nonce := make_gcm_nonce()
	encrypted_dek, err := encrypt(
		derived_key_encryption_key,
		new_dek_nonce,
		new_dek,
		lv.Vault.GetAad(),
	)

	if err != nil {
		return nil
	}

	lv.Vault.CryptoConfig.Key = encrypted_dek
	lv.Vault.CryptoConfig.Nonce = new_dek_nonce

	new_nonce := make_gcm_nonce()
	encrypted_data, err := encrypt(
		new_dek,
		new_nonce,
		data,
		nil,
	)

	if err != nil {
		return nil
	}

	lv.Vault.Data.Ciphertext = encrypted_data
	lv.Vault.Data.Nonce = new_nonce
	return nil
}

func (lv LocalVault) ReadData(passphrase []byte) ([]byte, error) {
	derived_key_encryption_key := derive_kek_from_passphrase(
		passphrase,
		lv.Vault.CryptoConfig.Salt,
	)

	data_encryption_key, err := decrypt(
		derived_key_encryption_key,
		lv.Vault.CryptoConfig.Nonce,
		lv.Vault.CryptoConfig.Key,
		lv.Vault.GetAad(),
	)

	if err != nil {
		return nil, err
	}

	data, err := decrypt(
		data_encryption_key,
		lv.Vault.Data.Nonce,
		lv.Vault.Data.Ciphertext,
		nil,
	)

	if err != nil {
		return nil, nil
	}

	return data, nil
}

func (lv *LocalVault) LoadFromFile(filename string) error {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var lvf LocalVaultFile
	if err := yaml.Unmarshal(raw, &lvf); err != nil {
		return err
	}

	// validate the file structure here
	lv.Vault = lvf
	return nil
}

func (lv LocalVault) WriteToFile(filename string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	if err := encoder.Encode(&lv.Vault); err != nil {
		return err
	}

	return nil
}

func (lv LocalVault) PrintYAML() error {
	out, err := yaml.Marshal(&lv.Vault)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(out)
	return err
}

type HexBytes []byte

func (h *HexBytes) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("expected hex string, got YAML node kind %d", value.Kind)
	}

	if value.Value == "" {
		*h = nil
		return nil
	}

	b, err := hex.DecodeString(value.Value)
	if err != nil {
		return fmt.Errorf("invalid hex value %q: %w", value.Value, err)
	}

	*h = HexBytes(b)
	return nil
}

func (h HexBytes) MarshalYAML() (any, error) {
	return hex.EncodeToString([]byte(h)), nil
}

func (lv *LocalVault) Init(passphrase []byte) error {
	lvf := LocalVaultFile{
		Version: V1String,
	}

	argon_salt := make_argon_salt()
	gcm_nonce := make_gcm_nonce()
	derived_key_encryption_key := derive_kek_from_passphrase(passphrase, argon_salt)
	random_dek := make_key()
	encrypted_dek, err := encrypt(derived_key_encryption_key, gcm_nonce, random_dek, lvf.GetAad())
	if err != nil {
		return err
	}

	default_data := []byte("")
	data_nonce := make_gcm_nonce()
	encrypted_data, err := encrypt(random_dek, data_nonce, default_data, nil)
	if err != nil {
		return err
	}

	lvf.CryptoConfig = LocalVaultCryptoConfig{
		Salt:  HexBytes(argon_salt),
		Key:   HexBytes(encrypted_dek),
		Nonce: HexBytes(gcm_nonce),
	}

	lvf.Data = LocalVaultData{
		Ciphertext: HexBytes(encrypted_data),
		Nonce:      HexBytes(data_nonce),
	}

	lv.Vault = lvf
	return nil
}

func derive_kek_from_passphrase(passphrase []byte, salt []byte) []byte {
	return argon2.IDKey(
		passphrase,
		salt,
		argonTime,
		argonMemoryKiB,
		argonParallelism,
		keyLength,
	)
}

const (
	argonTime        uint32 = 1
	argonMemoryKiB   uint32 = 64 * 1024
	argonParallelism uint8  = 4
	keyLength        uint32 = 32
)

func make_argon_salt() []byte {
	salt := make([]byte, 16)
	rand.Read(salt)

	return salt
}

func make_gcm_nonce() []byte {
	salt := make([]byte, 12) // recommended by someone i guess
	rand.Read(salt)

	return salt
}

func make_key() []byte {
	key := make([]byte, 32)
	rand.Read(key)

	return key
}

func encrypt(key []byte, nonce []byte, plaintext []byte, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return aesgcm.Seal(nil, nonce, plaintext, aad), nil
}

func decrypt(key []byte, nonce []byte, ciphertext []byte, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
