package main

// directory struct: {root}/{book title}/{chapter num}/{block num}
// when program begin, it will look through the dirs
// and make following structs.

type book struct {
	title string
	chapters []chaper
}

type chapter struct {
	num int
	name string
	blocks []block
}

// block is a string block of resonable length.
// anyway user will decide how long will it be.
type block struct {
	num int
	orig string
	trans string
}

