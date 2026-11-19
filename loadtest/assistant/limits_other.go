//go:build !unix

package main

func raiseFileLimit(int) {}
