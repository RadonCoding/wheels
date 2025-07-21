<div align="center">
    <img src="https://github.com/RadonCoding/wheels/blob/main/example.gif?raw=true" width="400" />
</div>
<br/>

Simple API to generate customizable animated spinning wheels as GIFs.

## Building

Check [.env.example](https://github.com/RadonCoding/wheels/blob/main/.env.example) for configuration.

1. `git clone https://github.com/RadonCoding/wheels.git`
2. `cd wheels`
3. `go build -o wheels`

## Usage

**Make a GET request to `/` with the following parameters:**

`options`- Comma-separated list of wheel labels (e.g. `a,b,c`)

`target`- Index of the option that should be the final result (zero-indexed)

`duration` - Duration of the GIF in seconds

`fps` - Frames per second for the GIF

## Example

`GET /?options=Red,Green,Blue&target=0&duration=10&fps=24`

This will return an animated GIF where the wheel spins for 10 seconds at 24 FPS and lands on "Red".

## Contributing

1. Fork it
2. Create your branch (`git checkout -b my-change`)
3. Commit your changes (`git commit -m "changed something"`)
4. Push to the branch (`git push origin my-change`)
5. Create new pull request

## License

This project is licensed under the MIT License.
