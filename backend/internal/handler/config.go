// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"english-app/internal/service"
)

type Config struct {
	Origin, StaticDir string
	SecureCookie      bool
}
type Server struct {
	Services *service.Services
	Config   Config
}
