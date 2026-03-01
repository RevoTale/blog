package web

//go:generate go run ../../framework/cmd/approutegen
//go:generate go run ../../framework/cmd/i18nkeygen -in i18n/messages/active.en.json -out i18n/keys_gen.go -pkg i18n
//go:generate go run ../../framework/cmd/templgen -base . -path components -path gen
