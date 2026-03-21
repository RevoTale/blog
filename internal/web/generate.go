package web

//go:generate go run github.com/RevoTale/no-js/framework/cmd/approutegen
//go:generate go run github.com/RevoTale/no-js/framework/cmd/i18nkeygen -in i18n/messages/active.en.json -out i18n/keys_gen.go -pkg i18n
//go:generate go run github.com/RevoTale/no-js/framework/cmd/templgen -base . -path components -path gen
