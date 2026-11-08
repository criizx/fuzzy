package fuzzy

// Default numerical resolution and tolerance.
const (
	DefaultAlphaLevels = 200
	DefaultGridPoints  = 1000
	DefaultEps         = 1e-9
)

// Config controls numerical resolution. Non-positive fields use defaults.
type Config struct {
	AlphaLevels int     // number of alpha steps for arithmetic and integration
	GridPoints  int     // x-grid resolution for membership-based metrics
	Eps         float64 // floating-point comparison tolerance
}

func DefaultConfig() Config {
	return Config{AlphaLevels: DefaultAlphaLevels, GridPoints: DefaultGridPoints, Eps: DefaultEps}
}

func (c Config) WithDefaults() Config {
	if c.AlphaLevels < 1 {
		c.AlphaLevels = DefaultAlphaLevels
	}
	if c.GridPoints < 1 {
		c.GridPoints = DefaultGridPoints
	}
	if c.Eps <= 0 {
		c.Eps = DefaultEps
	}
	return c
}
