//go:build !linux
// +build !linux

// Package main — interactive menu for non-Linux (macOS, Windows).
// Uses plain-text fallback since raw terminal is Linux-only.

package main

func showInteractiveMenu(items []menuItem) []menuItem {
	return showTextMenu(items)
}
