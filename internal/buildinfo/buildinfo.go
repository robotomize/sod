package buildinfo

const Graffiti = " _____  ___________ \n/  ___||  _  |  _  \\\n\\ `--. | | | | | | |\n `--. \\| | | | | | |\n/\\__/ /\\ \\_/ / |/ / \n\\____/  \\___/|___/  \n\n"

var (
	BuildTag string = "v0.0.0"
	Name     string = "SOD"
	Time     string = ""
)

type buildinfo struct{}

func (buildinfo) Tag() string {
	return BuildTag
}

func (buildinfo) Name() string {
	return Name
}

func (buildinfo) Time() string {
	return Time
}

var Info buildinfo
