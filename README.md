# wheels

**wheels** is a Go-based web service that dynamically generates animated "spin-the-wheel" GIFs via input parameters.

<br>

## Features

-   Fully animated spinning wheel exported as a GIF
    
-   Customize options, target result, duration, and FPS
    
-   Styled using a theme
    
-   Easy deployment as a standalone HTTP service

<br>

## Requirements

-   Go 1.22 or higher
    
<br>

## Installation

Clone the repository and build the project:

1. `git clone https://github.com/RadonCoding/wheels.git`
2. `cd wheels`
3. `go build -o wheels`

Then run it:

`./wheels` 

By default, the server will start on `localhost:8080`.

<br>

## Usage

**Make a GET request to `/` with the following query parameters:**


`options`- Comma-separated list of wheel labels (e.g. `a,b,c,d`)

`target`- Index of the option that should be the final result (0-based)

`duration` - Duration of the animation in seconds

`fps` - Frames per second for the GIF animation

---

### Example Request

`GET /?options=apple,banana,grape,pear&target=2&duration=5&fps=20` 

This will return an animated GIF where the wheel spins for 5 seconds at 20 FPS and lands on “grape”.

---

### cURL Example

`curl "http://localhost:8080/?options=red,blue,green,yellow&target=1&fps=15&duration=6" --output wheel.gif` 

---

<br>

## Theme

The wheel's appearance is controlled by the **Theme struct** in `wheel.go`, which defines all key visual elements, including colors and fonts. Each visual component (e.g. background, highlights, arrow) can be customized using RGBA values.

<br>

## License

This project is licensed under the MIT License.
