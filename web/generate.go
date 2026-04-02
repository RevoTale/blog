package web

//go:generate go run github.com/RevoTale/no-js/cmd/no-js gen routes -root ..
//go:generate go run github.com/RevoTale/no-js/cmd/i18nkeygen -in i18n/messages/active.en.json -out generated/i18nkeys/keys_gen.go -pkg i18nkeys
//go:generate go run github.com/RevoTale/no-js/cmd/templgen -base . -path components -path generated
