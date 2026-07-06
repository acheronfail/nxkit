package hacbrewpack

import _ "embed"

var (
	//go:embed assets/main.nso
	defaultExefsMain []byte
	//go:embed assets/main.npdm
	defaultExefsNPDM []byte
	//go:embed assets/NintendoLogo.png
	defaultNintendoLogo []byte
	//go:embed assets/StartupMovie.gif
	defaultStartupMovie []byte
	//go:embed assets/DefaultNSPImage.jpg
	defaultIcon []byte
	//go:embed assets/hacbrewpack.priv.pem
	defaultPrivateKeyPEM []byte
	//go:embed assets/hacbrewpack.pub.pem
	defaultPublicKeyPEM []byte
)
