# nview

CLI Tool for calculating N-view index of a vtuber.

## So what is N-view index?

N-view index is a metric used to measure the popularity of a vtuber. Simply speaking, it represents the number of digits in the CCV (Concurrent Viewers Count).

If a vtuber has a concurrent viewers of 9, then the N-view is 1, if 99, then 2, and so on.

## Features

- Fetches VTuber information from the VTStats API
- Computes the CCV and NView based on the VTuber's channel data
- Supports output in text or JSON format
- Configurable logging levels (debug, info, warn, error)

## Installation

1. Make sure you have Go installed on your system. You can download and install Go from the official website: <https://golang.org/>

2. Clone the repository or download the source code:

```bash
git clone https://github.com/amicus-veritatis/nview.git
```

3. Navigate to the project directory:

```bash
cd nview
```

4. Build the executable:

```bash
go build
```

## Usage

To use NView, run the executable with the following command:

```
./nview -name <name> [-output <output>] [-log <log>]
```

- `-name, -n <name>`: Specify the name of the VTuber you want to retrieve information for (required).
- `-output, -o <output>`: Choose the output format. Valid options are `text` (default) and `json`.
- `-log, -l <log>`: Set the log level. Valid options are `debug`, `info`, `warn` (default), and `error`.

Example:

```
./nview -name "Otonose Kanade" -output json -log error
```

Output:

```json
{"name":"Otonose Kanade","affiliation":"Hololive DEV_IS ReGLOSS","ccv":4961,"n_view":4}
```

## License

This project is licensed under the [MIT License](LICENSE).

## Contributing

Contributions are welcome! If you find any issues or have suggestions for improvements, please open an issue or submit a pull request.

## Acknowledgements

- [VTStats API](https://github.com/vtstats/server/): Provides the VTuber data used by NView.