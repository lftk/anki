# Anki

Anki is a Go library for reading and writing Anki collection files. It aims to provide a simple and intuitive API that follows idiomatic Go practices.

## Features

- **Full Read, Write, and Modify Capabilities:** Unlike many other libraries that are read-only or write-only, this library provides a complete solution for reading, writing, and modifying Anki collections.
- **Comprehensive Collection Management:** Programmatically manage your Anki collections, including decks, notes, notetypes, cards, and media files.
- **Seamlessly work with Anki Files:** Work with Anki's `.apkg` and `.colpkg` formats.
- **Flexibility:** Whether you're building tools to automate card creation, exporting your collection to a different format, or analyzing your study habits, this library provides the foundation you need.
- **Idiomatic Go:** The library is designed to feel natural for Go developers, with a clean and easy-to-use API.

## Installation

```bash
go get github.com/lftk/anki
```

## API Reference

For a complete list of available functions and types, please refer to the [Go documentation](https://pkg.go.dev/github.com/lftk/anki).

## Command-Line Tool

For a simple command-line tool to unpack `.apkg` and `.colpkg` files, check out the companion project: [anki-unpkg](https://github.com/lftk/anki-unpkg).

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue.

## License

This project is licensed under the AGPL-3.0 License. See the [LICENSE](LICENSE) file for details.

## Acknowledgements

This project is heavily inspired by and based on the official [Anki](https://apps.ankiweb.net/) ecosystem. A big thank you to the Ankitects team and all contributors for their tremendous work on the original software.
