.PHONY: stage audit test build package verify source clean
stage audit test build package verify source:
	python3 scripts/ztarc.py $@
clean:
	python3 scripts/ztarc.py clean
