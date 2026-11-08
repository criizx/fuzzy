// Package fuzzy implements convex fuzzy values through membership functions and alpha cuts.
//
// LogNormal derives a finite fuzzy model from distribution.LogNormal by normalizing
// its density to the mode and applying an explicit relative-density cutoff.
// Probability densities, CDFs, and probabilistic moments belong to package distribution;
// possibilistic moments, interval arithmetic, comparisons, and distances belong here.
//
// Arithmetic follows the extension principle over sampled alpha cuts. It does not
// calculate probability distributions of independent sums or preserve operand dependence.
// Nonlinear results are represented by nested Cuts rather than forced into a lognormal family.
package fuzzy
