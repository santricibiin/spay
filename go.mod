module github.com/hirotomasato/paygatemeapp

go 1.26.2

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/hirotomasato/paygateme v0.1.0
	github.com/jackc/pgx/v5 v5.11.0
	github.com/makiuchi-d/gozxing v0.1.1
	golang.org/x/crypto v0.57.0
	golang.org/x/image v0.46.0
)

require (
	github.com/dchest/captcha v1.1.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/wenlng/go-captcha v1.2.5 // indirect
	github.com/wenlng/go-captcha/v2 v2.0.5 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)

replace github.com/hirotomasato/paygateme => ../paygateme
