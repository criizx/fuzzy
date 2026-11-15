TEX = report

.PHONY: all xelatex clean
all:
	tectonic $(TEX).tex

xelatex:
	xelatex -interaction=nonstopmode -halt-on-error $(TEX).tex
	xelatex -interaction=nonstopmode -halt-on-error $(TEX).tex

clean:
	rm -f $(TEX).aux $(TEX).log $(TEX).out $(TEX).toc
